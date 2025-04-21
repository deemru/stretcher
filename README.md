# Stretcher

Stretcher is a minimal request throttling proxy server written in Go (as a successor to [Stretcher-PHP](https://github.com/deemru/Stretcher-PHP)).

- Serializes requests per IP
- Throttles requests based on their consumption
- Configurable window and target
- Easy to deploy and monitor
- Built-in debug logging

## Basic usage

```bash
go run stretcher.go --listen=127.0.0.1:8080 --upstream=127.0.0.1:80 --timeout=12 --window=12 --target=4 --concurrency=64 --maxbytes=65536 --debug=true
```

## Configuration

| Option | Description | Default Value |
|--------|-------------|---------------|
| `--listen` | Listen address and port | 127.0.0.1:8080 |
| `--upstream` | Upstream server address and port | 127.0.0.1:80 |
| `--timeout` | Request timeout in seconds | 12 |
| `--window` | Time window for rate limiting in seconds | 12 |
| `--target` | Target requests per window | 4 |
| `--concurrency` | Maximum concurrent requests per IP | 64 |
| `--maxbytes` | Maximum request body size in bytes | 65536 |
| `--debug` | Enable debug logging | true |

Example usage:
```bash
go run stretcher.go --listen=127.0.0.1:8080 --upstream=127.0.0.1:80 --timeout=12 --window=12 --target=4 --concurrency=64 --maxbytes=65536 --debug=true
```

## Installation

### Manual installation

```bash
# Install Go
apt update
apt install golang-go git -y

# Set up Go environment
mkdir -p ~/go
export GOPATH=~/go
export PATH=$PATH:$GOPATH/bin
```

### Create user and setup directory
```bash
useradd -m -s /bin/bash stretcher
mkdir /stretcher
chown -R stretcher /stretcher
```

### Clone repository
```bash
sudo -u stretcher bash -c "cd /stretcher && git clone https://github.com/deemru/stretcher.git"
```

## Building

### Build the binary
```bash
cd /stretcher/stretcher
go build -o stretcher
```

## Running

### Start the service
```bash
sudo -u stretcher bash -c "cd /stretcher/stretcher && ./stretcher --upstream=127.0.0.1:80 --debug=true"
```

### For automatic startup on boot
```bash
echo "@reboot stretcher cd /stretcher/stretcher && ./stretcher --upstream=127.0.0.1:80 --debug=true | systemd-cat -t Stretcher &" >> /etc/crontab
```

## Monitoring

### View the logs
```bash
journalctl -t Stretcher -f -n 100
```