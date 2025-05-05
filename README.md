# Stretcher

Stretcher is a minimal request throttling proxy server written in Go (as a successor to [Stretcher-PHP](https://github.com/deemru/Stretcher-PHP)).

- Serializes requests per IP
- Throttles requests based on their consumption
- Configurable window and target
- Easy to deploy and monitor
- Built-in debug logging

## Installation

### Option 1: Using Debian Package (Recommended)

```bash
curl -L -o stretcher.deb https://github.com/deemru/stretcher/releases/latest/download/stretcher_linux_amd64.deb
sudo dpkg -i stretcher.deb
```

The Debian package automatically:
- Creates a stretcher user
- Installs the binary to /usr/bin/stretcher
- Sets up a systemd service that starts automatically

To change the configuration:
```bash
sudo nano /etc/systemd/system/stretcher.service
sudo systemctl daemon-reload
sudo systemctl restart stretcher.service
```

### Option 2: Manual Installation

1. Create user and directory:
```bash
sudo useradd -m -s /bin/bash stretcher && sudo mkdir /stretcher && sudo chown -R stretcher /stretcher
```

2. Download and install the binary:
```bash
sudo rm -rf /stretcher/* && sudo curl -L -o /stretcher/stretcher https://github.com/deemru/stretcher/releases/latest/download/stretcher-linux-amd64 && sudo chmod +x /stretcher/stretcher
```

3. Set up autostart via crontab:
```bash
sudo nano /etc/crontab
```
Add this line:
```
@reboot stretcher systemd-cat -t Stretcher -- /stretcher/stretcher --upstream 127.0.0.1:80
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
stretcher --listen=127.0.0.1:8080 --upstream=127.0.0.1:80 --timeout=12 --window=12 --target=4 --concurrency=64 --maxbytes=65536 --debug=true
```

## Monitoring

### View the logs

For Debian package installation:
```bash
journalctl -u stretcher.service -f -n 100
```

For manual installation:
```bash
journalctl -t Stretcher -f -n 100
```

### Check service status (Debian package method)
```bash
sudo systemctl status stretcher
```

## Building from Source

If you want to build the binary yourself:

```bash
# Install Go
sudo apt update
sudo apt install golang-go git -y

# Clone the repository
git clone https://github.com/deemru/stretcher.git
cd stretcher

# Build the binary
go build -trimpath -ldflags "-s -w" -o stretcher stretcher.go
```