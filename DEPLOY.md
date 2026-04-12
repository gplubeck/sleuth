# Deploying Sleuth

## Quick start

### 1. Create a dedicated user

Sleuth does not need root. Run it as its own user with no login shell:

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin sleuth
```

### 2. Set up the install directory

Templates and CSS are embedded in the binary. Sleuth only needs `config.toml`
and a writable directory for `.sleuth.bin` (uptime history) at runtime.

```bash
sudo mkdir -p /opt/sleuth/bin
```

Copy the files:

```bash
sudo cp bin/sleuth        /opt/sleuth/bin/sleuth
sudo cp config.toml       /opt/sleuth/config.toml
```

Set ownership so the `sleuth` user can write `.sleuth.bin`:

```bash
sudo chown -R sleuth:sleuth /opt/sleuth
sudo chmod 750 /opt/sleuth
```

### 3. Edit the config

```bash
sudo nano /opt/sleuth/config.toml
```

Minimum required fields per service:

```toml
[server]
port         = 5000
storage_type = "memory"
log_level    = "warn"
theme        = "material_dark.css"
title        = "Service Status"

[[service]]
id           = 1
service_name = "My Service"
address      = "myhost.local:443"
protocol_str = "TCP"   # TCP, UDP, or Test
timer        = 30      # probe interval in seconds
MaxHistory   = 100     # number of checks to keep
```

### 4. Install and enable the systemd unit

```bash
sudo cp sleuth.service /etc/systemd/system/sleuth.service
sudo systemctl daemon-reload
sudo systemctl enable --now sleuth
```

Check that it started cleanly:

```bash
sudo systemctl status sleuth
sudo journalctl -u sleuth -f
```

---

## TLS (HTTPS)

### Using your own certificate

Set `cert_file` and `cert_key` in `config.toml`:

```toml
[server]
port      = 443
cert_file = "/opt/sleuth/server.pem"
cert_key  = "/opt/sleuth/server-key.pem"
```

Make the key readable only by the `sleuth` user:

```bash
sudo chown sleuth:sleuth /opt/sleuth/server-key.pem
sudo chmod 600 /opt/sleuth/server-key.pem
```

### Generating a self-signed cert (development / internal use)

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout server-key.pem -out server.pem \
  -days 365 -nodes \
  -subj "/CN=yourhostname" \
  -addext "subjectAltName=DNS:yourhostname,IP:192.168.1.x"
```

### Binding to port 443 without root

By default, ports below 1024 require root. Two clean options:

**Option A — `CAP_NET_BIND_SERVICE`** (grant the binary the capability):

```bash
sudo setcap 'cap_net_bind_service=+ep' /opt/sleuth/bin/sleuth
```

**Option B — Reverse proxy** (recommended for production): run Sleuth on a high
port and put nginx or Caddy in front.

---

## Reverse proxy

### nginx

```nginx
server {
    listen 443 ssl;
    server_name status.example.com;

    ssl_certificate     /etc/ssl/certs/status.example.com.pem;
    ssl_certificate_key /etc/ssl/private/status.example.com-key.pem;

    location / {
        proxy_pass http://127.0.0.1:5000;

        # Required for Server-Sent Events
        proxy_buffering       off;
        proxy_cache           off;
        proxy_read_timeout    3600s;
        proxy_set_header Host $host;
    }
}
```

> **Important:** `proxy_buffering off` is required. Without it nginx will buffer
> the SSE stream and updates will never reach the browser.

### Caddy

```caddy
status.example.com {
    reverse_proxy localhost:5000 {
        flush_interval -1
    }
}
```

---

## Managing the service

| Task | Command |
|---|---|
| Start | `sudo systemctl start sleuth` |
| Stop | `sudo systemctl stop sleuth` |
| Restart | `sudo systemctl restart sleuth` |
| Reload config (no restart) | `sudo systemctl reload sleuth` |
| View logs | `sudo journalctl -u sleuth -f` |
| View last 100 log lines | `sudo journalctl -u sleuth -n 100` |

### Reloading config without a restart

After editing `config.toml`, send SIGHUP:

```bash
sudo systemctl reload sleuth
```

This re-parses the config and reconciles the service list:
- New services start being monitored immediately.
- Removed services stop being monitored immediately.
- Existing services have their config fields (name, address, timer) updated while
  keeping their uptime history.

> Note: server-level settings (`port`, `cert_file`, `cert_key`) require a full
> restart to take effect.

---

## Starting without history

If the on-disk history (`.sleuth.bin`) is stale or corrupt, start fresh:

```bash
sudo systemctl edit sleuth
```

Add an override:

```ini
[Service]
ExecStart=
ExecStart=/opt/sleuth/bin/sleuth --no-history
```

Then `sudo systemctl restart sleuth`. Remove the override once the file is clean.

Or delete the file directly:

```bash
sudo systemctl stop sleuth
sudo rm /opt/sleuth/.sleuth.bin
sudo systemctl start sleuth
```

---

## Alpine Linux (OpenRC)

An OpenRC example script is provided in `openrc_example`. Copy it to
`/etc/init.d/sleuth`, set `owning_dir` to your install path, then:

```bash
rc-update add sleuth default
rc-service sleuth start
```
