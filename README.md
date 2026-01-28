# Integration Suricata + nDPI

Integration Suricata + nDPI automates configuration management for Suricata
Deployments that use the nDPI plugin. It provides a host-side agent (Unix
socket HTTP API) to enable/disable the plugin safely and an integration
service for reloadable updates without downtime.

## Overview

This repository contains:

- **Integration service**: validates local inputs, renders and writes
  `suricata.yaml` atomically, and issues reload/reconfigure via `suricatasc`
  when possible.
- **Host Agent**: runs on the Suricata host, edits the active Suricata config
  to enable/disable the nDPI plugin, and restarts Suricata via systemd only
  when necessary.

## Requirements

- Go 1.21+.
- Suricata installed and managed by systemd (service name: usually
  `suricata`, configure via `system.suricata_service`).
- Suricata control socket available (one of `suricata.socket_candidates`,
  e.g. `/run/suricata/suricata-command.socket`).
- Host Agent installed on the host (recommended `/usr/local/bin/host-agent`).
- Host Agent config present (recommended
  `/etc/integration-suricata-ndpi/config.yaml`).
- nDPI plugin file exists on host (`paths.ndpi_plugin_path`, e.g.
  `/usr/local/lib/suricata/ndpi.so`).
- Permission to restart Suricata via systemctl (Host Agent runs as root).

### Supported versions

- Suricata 8.0.x (built with `--enable-ndpi`).
- nDPI 4.14.

> **Note**: Package-manager installs may ship non-nDPI builds. Prefer a stable
> source tarball and an explicit configure step.

## Quick Start

### Build

```bash
go build -o bin/integration ./cmd/integration
go build -o bin/host-agent ./cmd/host-agent
```

### Run

```bash
sudo ./bin/integration run --config config/config.yaml
sudo ./bin/host-agent serve --config config/config.yaml --sock /run/ndpi-agent.sock
```

## Configuration

- Main config: `config/config.yaml`.
- Suricata template: `config/suricata.yaml.tpl`.

### Minimal required fields

- `paths.ndpi_rules_local` - local nDPI rules directory.
- `paths.suricata_template` - `suricata.yaml.tpl` template.
- `paths.suricatasc` - path to `suricatasc`.
- `suricata.socket_candidates` - Unix-socket path candidates.
- `suricata.config_candidates` - `suricata.yaml` path candidates.
- `reload.command`, `reload.timeout` - reload/reconfigure parameters.

> **NEEDS CLARIFICATION**: provide a full example config and document all
> optional fields (timeouts, HTTP listen address, systemd paths).

## nDPI plugin activation notes

- nDPI plugin activation requires a Suricata process restart.
- The integration service does **not** restart Suricata to toggle the plugin.
  Enable/disable is delegated to the Host Agent.

## Installing Suricata 8.0.x with nDPI 4.14 (Debian/Ubuntu)

### 1. Install build dependencies

```bash
sudo apt-get update
sudo apt-get install -y \
  build-essential git \
  autoconf automake libtool pkg-config gettext \
  flex bison \
  libpcap-dev libjson-c-dev libnuma-dev \
  libpcre2-dev libmaxminddb-dev librrd-dev \
  libyaml-dev libjansson-dev libmagic-dev \
  rustc cargo
```

### 2. Build and install nDPI 4.14

```bash
mkdir -p ~/src && cd ~/src

git clone https://github.com/ntop/nDPI.git
cd nDPI

git checkout ndpi-4.14

./autogen.sh
./configure --with-only-libndpi
make -j"$(nproc)"
sudo make install
sudo ldconfig
```

This installs the nDPI library into default system paths (for example
`/usr/local/lib`, `/usr/local/include`).

### 3. Download and build Suricata 8.0.2 with nDPI support

```bash
cd ~/src

wget https://www.openinfosecfoundation.org/download/suricata-8.0.2.tar.gz
tar xzf suricata-8.0.2.tar.gz
cd suricata-8.0.2
```

Configure Suricata with nDPI enabled (adjust the nDPI path if needed):

```bash
./configure \
  --enable-ndpi \
  --with-ndpi=$HOME/src/nDPI \
  --prefix=/usr \
  --sysconfdir=/etc \
  --localstatedir=/var
```

Build and install:

```bash
make -j"$(nproc)"
sudo make install
sudo ldconfig
```

After this step you should have:

- `suricata` under `/usr/bin`.
- `/etc/suricata` configuration directory.
- `ndpi.so` under `/usr/lib/suricata` or `/usr/local/lib/suricata`.

### 4. Enable the nDPI plugin in Suricata configuration

Ensure the plugin section contains the nDPI shared object:

```yaml
plugins:
  - /usr/lib/suricata/ndpi.so
```

Adjust the path if `ndpi.so` is installed under `/usr/local/lib/suricata`.

### 5. Basic verification

```bash
suricata --build-info

ls -l /usr/lib/suricata/ndpi.so || \
  ls -l /usr/local/lib/suricata/ndpi.so

sudo suricata -c /etc/suricata/suricata.yaml -T
```

## Host Agent

### What the Host Agent does

- Modifies active `suricata.yaml` by commenting/uncommenting the `ndpi.so`
  plugin line within the `plugins:` section.
- Writes the updated configuration atomically.
- Restarts Suricata via systemd only if a state change is needed.

### Why a Host Agent is required

In typical deployments the integration service runs in a container, while
Suricata runs on the host. Editing `/etc/suricata/suricata.yaml` and restarting
Suricata require host-level privileges and systemd access, which is not granted
to the container.

### API overview (Unix socket)

- `GET /health` - liveness probe.
- `POST /suricata/ensure` - ensure Suricata is running and socket is reachable.
- `GET /ndpi/status` - current state from config contents.
- `POST /ndpi/enable` - enable nDPI plugin and restart Suricata.
- `POST /ndpi/disable` - disable nDPI plugin and restart Suricata.
- `POST /suricata/reload` - reload rules via `suricatasc`.

### Example usage

Start Host Agent:

- `sudo ./bin/host-agent serve --config config/config.yaml --sock /run/ndpi-agent.sock`

Enable nDPI:

- `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/enable`

Disable nDPI:

- `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/disable`

### Operational notes

Enabling/disabling the plugin is a restart-level change and may briefly
interrupt traffic inspection during Suricata restart. Reload operations
(`suricatasc`, ExecReload) are suitable for reloadable changes but are not
reliable for dynamic plugin (un)loading.

## Host Agent deployment (systemd)

> Assumption: `host-agent` is installed at `/usr/local/bin/host-agent`, and
> config is at `/etc/integration-suricata-ndpi/config.yaml`.

Install unit files:

```bash
sudo install -D -m 0644 deploy/systemd/ndpi-agent.socket  /etc/systemd/system/ndpi-agent.socket
sudo install -D -m 0644 deploy/systemd/ndpi-agent.service /etc/systemd/system/ndpi-agent.service
```

Reload systemd and start socket activation:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ndpi-agent.socket
```

Optional: always-on mode (run service even without incoming requests):

```bash
sudo systemctl enable --now ndpi-agent.service
```

Update binary:

```bash
sudo systemctl stop ndpi-agent.service ndpi-agent.socket
sudo install -m 0755 bin/host-agent /usr/local/bin/host-agent
sudo systemctl start ndpi-agent.socket
```

## Integration service API (TCP)

Default listen address: `http.addr` (example `:8080`).

### Health

```bash
curl http://localhost:8080/health
```

### Plan / Apply

Plan (dry-run, no changes):

```bash
curl http://localhost:8080/plan
```

Reconcile (patch config, validate, restart Suricata if needed): - WIP

```bash
curl -X POST http://localhost:8080/plan
```

Apply (reload rules via `suricatasc`, ensures Suricata via Host Agent first):

```bash
curl -X POST http://localhost:8080/apply
```

### nDPI toggle via integration (delegates to Host Agent)

```bash
curl -X POST http://localhost:8080/ndpi/enable
curl -X POST http://localhost:8080/ndpi/disable
```

## Operational commands

### Check service/socket state

```bash
sudo systemctl status ndpi-agent.socket
sudo systemctl status ndpi-agent.service
```

## Rules update (no Suricata restart)

Suricata rules can be reloaded without restarting Suricata using
`suricatasc reload-rules`.

### Manual reload (recommended for debugging)

```bash
sudo /usr/local/bin/suricatasc -c reload-rules /run/suricata/suricata-command.socket
```

### Reload via Host Agent

```bash
sudo curl -sS -X POST --unix-socket /run/ndpi-agent.sock http://localhost/suricata/reload
```

## Troubleshooting

> **NEEDS CLARIFICATION**: provide common failure modes and remediation steps
> (socket missing, permissions, systemd restart failures).
