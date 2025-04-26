package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var Version = "Go"

type requestEvent struct {
	req      *http.Request
	w        http.ResponseWriter
	done     chan struct{}
	enqueued time.Time
}

type IPState struct {
	queue  chan *requestEvent
	queued int
	ttlast time.Time
	cclast float64
	ddlast float64
	ip     string
	mu     sync.Mutex
}

var (
	ipStates     = make(map[string]*IPState)
	mapLock      = &sync.RWMutex{}
	listen       string
	upstream     string
	upstreamHost string
	upstreamURL  *url.URL
	timeout      int
	ttwindow     float64
	cctarget     float64
	concurrency  int
	maxbytes     int
	quant        float64
	debug        bool

	client = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(timeout) * time.Second,
			}).DialContext,
			DisableCompression: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

type jsonMethod struct {
	Method string `json:"method"`
}

func main() {
	flag.StringVar(&listen, "listen", "127.0.0.1:8080", "listen address")
	flag.StringVar(&upstream, "upstream", "127.0.0.1:80", "upstream base URL (e.g. http://host:port)")
	flag.IntVar(&timeout, "timeout", 12, "upstream timeout")
	flag.Float64Var(&ttwindow, "window", 12.0, "sliding window seconds")
	flag.Float64Var(&cctarget, "target", 4.0, "target in seconds")
	flag.IntVar(&concurrency, "concurrency", 64, "maximum concurrent requests per IP")
	flag.IntVar(&maxbytes, "maxbytes", 65536, "max body bytes")
	flag.BoolVar(&debug, "debug", true, "enable debug logging")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime)
	log.Printf("Stretcher (%s)", Version)
	log.Printf("Stretching %s -> %s (timeout: %d, window: %.0f, target: %.0f, concurrency: %d, maxbytes: %d, debug: %t)",
		listen, upstream, timeout, ttwindow, cctarget, concurrency, maxbytes, debug)

	quant = ttwindow / cctarget

	upstream = strings.TrimPrefix(upstream, "http://")
	if i := strings.IndexByte(upstream, '/'); i >= 0 {
		upstreamHost = upstream[:i]
	} else {
		upstreamHost = upstream
		upstream = upstream + "/"
	}

	if !strings.HasPrefix(upstream, "http://") {
		upstream = "http://" + upstream
	}

	var err error
	upstreamURL, err = url.Parse(upstream)
	if err != nil {
		log.Fatalf("Failed to parse upstream URL: %v", err)
	}

	client.Timeout = time.Duration(timeout) * time.Second

	go cleanupIPStates()

	server := &http.Server{
		Addr:    listen,
		Handler: http.HandlerFunc(enqueueHandler),
	}

	log.Fatal(server.ListenAndServe())
}

func cleanupIPStates() {
	window := time.Duration(ttwindow * float64(time.Second))
	ticker := time.NewTicker(window)
	defer ticker.Stop()

	for range ticker.C {
		if len(ipStates) > 0 {
			now := time.Now()
			var toRemove []string

			mapLock.RLock()
			for ip, state := range ipStates {
				if len(state.queue) == 0 && now.Sub(state.ttlast) > window {
					toRemove = append(toRemove, ip)
				}
			}
			mapLock.RUnlock()

			if len(toRemove) > 0 {
				mapLock.Lock()
				for _, ip := range toRemove {
					if state, exists := ipStates[ip]; exists {
						if len(state.queue) == 0 && now.Sub(state.ttlast) > window {
							close(state.queue)
							delete(ipStates, ip)
						}
					}
				}
				mapLock.Unlock()
			}
		}
	}
}

func formatURI(req *http.Request) string {
	uri := req.URL.Path
	if uri == "" {
		uri = "/"
	}
	if req.URL.RawQuery != "" {
		uri += "?" + req.URL.RawQuery
	}

	if req.Method == http.MethodPost {
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil && len(bodyBytes) > 0 {
				var jsonReq jsonMethod
				if err := json.Unmarshal(bodyBytes, &jsonReq); err == nil && jsonReq.Method != "" {
					uri = jsonReq.Method
				} else {
					uri += " (POST)"
				}
			} else {
				uri += " (POST)"
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		} else {
			uri += " (POST)"
		}
	}

	return uri
}

func logRequestComplete(state *IPState, uri string, cc float64, statusCode int) {
	log.Printf("%s: %d (%d/%d/%d/%d): %s",
		state.ip,
		statusCode,
		state.queued,
		int(1000*state.cclast),
		int(1000*state.ddlast),
		int(1000*cc),
		uri)
}

func enqueueHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		req.Body = http.MaxBytesReader(w, req.Body, int64(maxbytes))
	case http.MethodGet, http.MethodOptions:
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(req)
	state := getIPState(clientIP)

	state.mu.Lock()
	if state.queued >= concurrency {
		state.mu.Unlock()
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	state.queued++
	state.mu.Unlock()

	e := &requestEvent{
		req:      req,
		w:        w,
		done:     make(chan struct{}),
		enqueued: time.Now(),
	}

	state.queue <- e

	<-e.done

	state.mu.Lock()
	state.queued--
	state.mu.Unlock()
}

func getClientIP(req *http.Request) string {
	cfConnectingIP := req.Header.Get("Cf-Connecting-Ip")
	if cfConnectingIP != "" {
		return cfConnectingIP
	}

	forwarded := req.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return ip
}

func getIPState(ip string) *IPState {
	mapLock.RLock()
	state, ok := ipStates[ip]
	mapLock.RUnlock()

	if ok {
		return state
	}

	mapLock.Lock()
	defer mapLock.Unlock()

	state, ok = ipStates[ip]
	if ok {
		return state
	}

	state = &IPState{
		queue:  make(chan *requestEvent, concurrency),
		queued: 0,
		ttlast: time.Time{},
		cclast: 0.0,
		ddlast: 0.0,
		ip:     ip,
		mu:     sync.Mutex{},
	}
	ipStates[ip] = state
	go ipWorker(state)

	return state
}

func ipWorker(state *IPState) {
	for ev := range state.queue {
		handleEvent(state, ev)
		close(ev.done)
	}
}

func handleEvent(state *IPState, ev *requestEvent) {
	req := ev.req
	w := ev.w
	var uri string
	if debug {
		uri = formatURI(req)
	}
	ttnow := time.Now()

	var ttdiff float64
	var delay float64

	if state.ttlast.IsZero() {
		ttdiff = 0
	} else {
		ttdiff = ttnow.Sub(state.ttlast).Seconds()
	}

	if ttdiff < ttwindow {
		fading := 1 - (ttdiff / ttwindow)
		state.cclast *= fading
		state.ddlast *= fading

		target := state.cclast*quant - state.cclast

		if ttdiff > target {
			delay = 0
		} else {
			delay = target - ttdiff
			if delay > ttwindow {
				delay = ttwindow
			}
		}

		delay = (state.ddlast + delay) / 2
		state.ddlast = delay
	} else {
		state.cclast = 0
		delay = 0
		state.ddlast = 0
	}

	state.ttlast = ttnow

	if delay > 0.001 {
		timer := time.NewTimer(time.Duration(delay * float64(time.Second)))

		select {
		case <-timer.C:
		case <-req.Context().Done():
			if !timer.Stop() {
				<-timer.C
			}
			if debug {
				logRequestComplete(state, uri, 0, 0)
			}
			return
		}
	}

	ttaction := time.Now()
	statusCode := proxyUpstream(w, req)

	ttnow = time.Now()
	cc := ttnow.Sub(ttaction).Seconds()

	ttdiff = ttnow.Sub(state.ttlast).Seconds()
	if ttdiff < ttwindow {
		fading := 1 - (ttdiff / ttwindow)
		state.cclast *= fading
		state.ddlast *= fading
	} else {
		state.cclast = 0
		state.ddlast = 0
	}

	state.cclast += cc
	state.ttlast = ttnow

	if debug {
		logRequestComplete(state, uri, cc, statusCode)
	}
}

func proxyUpstream(w http.ResponseWriter, req *http.Request) int {
	reqURL := *upstreamURL

	reqURL.Path = req.URL.Path
	reqURL.RawQuery = req.URL.RawQuery

	upstreamReq, err := http.NewRequestWithContext(req.Context(), req.Method, reqURL.String(), req.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return http.StatusInternalServerError
	}

	upstreamReq.Header = req.Header.Clone()
	upstreamReq.Host = upstreamHost

	resp, err := client.Do(upstreamReq)

	if err != nil {
		if req.Context().Err() != nil {
			return 0
		}

		var statusCode int

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			statusCode = http.StatusRequestTimeout
			http.Error(w, "Upstream Timeout", statusCode)
		} else {
			statusCode = http.StatusServiceUnavailable
			http.Error(w, "Upstream Error", statusCode)
		}
		return statusCode
	}

	defer resp.Body.Close()

	header := w.Header()
	for k, v := range resp.Header {
		header[k] = v
	}

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
	return resp.StatusCode
}
