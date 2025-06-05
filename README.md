# Stretcher

Stretcher is a minimal request throttling proxy server written in Go (as a successor to [Stretcher-PHP](https://github.com/deemru/Stretcher-PHP)).

- Serializes requests per IP
- Throttles requests based on their consumption
- Configurable window and target
- Easy to deploy and monitor
- Built-in debug logging

## Installation

Download and install the latest Stretcher release for Linux (AMD64):

```bash
curl -L -o stretcher.deb https://github.com/deemru/stretcher/releases/latest/download/stretcher_linux_amd64.deb \
    && dpkg -i stretcher.deb \
    && unlink stretcher.deb \
    && service stretcher status
```

- Downloads the latest `.deb` package.
- Installs it using `dpkg`.
- Deletes the downloaded package.
- Checks the status of the `stretcher` service.

The Debian package automatically:
- Creates a stretcher user
- Installs the binary to /usr/bin/stretcher
- Sets up a systemd service that starts automatically

## Service Configuration

Edit the systemd service file for Stretcher and apply changes:

```bash
nano /etc/systemd/system/stretcher.service \
    && systemctl daemon-reload \
    && systemctl restart stretcher.service \
    && journalctl -u stretcher -f -n 100
```

- Opens the service file for editing.
- Reloads systemd to apply changes.
- Restarts the Stretcher service.
- Streams the last 100 log entries and follows the log output.

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

### Example Configuration

Basic usage with custom settings:
```bash
stretcher --listen=0.0.0.0:8080 --upstream=127.0.0.1:80 --window=10 --target=6
```

Full configuration example:
```bash
stretcher --listen=127.0.0.1:8080 --upstream=127.0.0.1:80 --timeout=12 --window=12 --target=4 --concurrency=64 --maxbytes=65536 --debug=true
```

## Building from Source

Build Stretcher from source code:

```bash
# Install dependencies
sudo apt update && sudo apt install golang-go git -y

# Clone and build
git clone https://github.com/deemru/stretcher.git \
    && cd stretcher \
    && go build -trimpath -ldflags "-s -w" -o stretcher stretcher.go
```