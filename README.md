# Integration Suricata + nDPI

Integration Suricata + nDPI automates configuration management for Suricata deployments that use the nDPI plugin.

## Key capabilities

- Generates `suricata.yaml` from a template (with environment variable rendering) and writes it atomically to prevent partial/invalid config updates.
- Applies reloadable Suricata changes using best-effort `suricatasc` reload/reconfigure, avoiding service downtime when restart is not required.
- Validates local resources and nDPI-related inputs (plugin `ndpi.so`, rules directory, template integrity, `suricatasc` binary) before applying changes.
- Verifies Suricata availability via the Unix control socket (selecting a working socket candidate, not just an existing path).
- Provides a host-side agent API over a Unix socket to enable/disable the nDPI plugin by editing the active Suricata config and restarting Suricata via systemd only when a state change is needed.

## Important note about nDPI plugin activation

- nDPI plugin activation requires a Suricata process restart.
- The integration service itself does **not** restart Suricata to toggle the plugin. It only performs reloadable updates. Plugin enable/disable is delegated to the Host Agent.

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

### Configuration

- The configuration is defined by a YAML file (example: `config/config.yaml`).
- The Suricata template is stored in `config/suricata.yaml.tpl`.

### Minimally important fields

- `paths.ndpi_rules_local` — local nDPI rules directory.
- `paths.suricata_template` — `suricata.yaml.tpl` template.
- `paths.suricatasc` — path to `suricatasc`.
- `suricata.socket_candidates` — Unix-socket path candidates.
- `suricata.config_candidates` — `suricata.yaml` path candidates.
- `reload.command`, `reload.timeout` — best-effort reload/reconfigure parameters.

## Installing Suricata 8.0.x with nDPI 4.14 (Debian/Ubuntu)

This project assumes:

- Suricata 8.0.x built with nDPI support (`--enable-ndpi`).
- nDPI 4.14 installed on the host.

> Note: Package-manager installs (for example `apt install suricata`) may provide development or non‑nDPI builds. Prefer a stable source tarball and an explicit configure step.

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

# Checkout a tested release
git checkout ndpi-4.14   # adjust to the exact tag you use

./autogen.sh
./configure --with-only-libndpi
make -j"$(nproc)"
sudo make install
sudo ldconfig
```

This installs the nDPI library into the default system paths (for example `/usr/local/lib`, `/usr/local/include`).

### 3. Download and build Suricata 8.0.2 with nDPI support

```bash
cd ~/src

wget https://www.openinfosecfoundation.org/download/suricata-8.0.2.tar.gz
tar xzf suricata-8.0.2.tar.gz
cd suricata-8.0.2
```

Configure Suricata with nDPI enabled (adjust the nDPI path if you cloned it elsewhere):

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

- `suricata` installed under `/usr/bin`.
- Configuration directory under `/etc/suricata`.
- `ndpi.so` installed under `/usr/lib/suricata` or `/usr/local/lib/suricata`.

### 4. Enable the nDPI plugin in Suricata configuration

In your active Suricata configuration (or in the template used by this project), ensure the plugin section contains the nDPI shared object:

```yaml
plugins:
  - /usr/lib/suricata/ndpi.so
```

Adjust the path if `ndpi.so` is installed under `/usr/local/lib/suricata` instead.

### 5. Basic verification

```bash
suricata --build-info

ls -l /usr/lib/suricata/ndpi.so || \
  ls -l /usr/local/lib/suricata/ndpi.so

sudo suricata -c /etc/suricata/suricata.yaml -T
```

If the config test passes and `ndpi.so` is found, the Suricata + nDPI environment is ready to be managed by the integration service and Host Agent.

## nDPI plugin enable/disable (Host Agent)

Because Suricata must be restarted to (un)load the nDPI plugin shared object (`ndpi.so`), the integration service does not perform plugin toggling directly. Instead, plugin enable/disable is delegated to a Host Agent running on the Suricata host.

### What the Host Agent does

- Modifies the active Suricata configuration (`suricata.yaml`) by commenting/uncommenting the `ndpi.so` plugin line within the `plugins:` section.
- Writes the updated configuration atomically to avoid partial writes and corrupted configs.
- Restarts Suricata via systemd to apply the change reliably.

### Why a Host Agent is required

In typical deployments, the integration service runs in a container, while Suricata runs on the host. Restarting Suricata and editing `/etc/suricata/suricata.yaml` requires host-level privileges and access to systemd, which is intentionally not granted to the containerized integration service.

### API overview (Unix socket)

The Host Agent exposes a small HTTP API over a Unix socket:

- `GET /health` — liveness probe.
- `POST /suricata/ensure` — ensures Suricata is running and the control socket is reachable.
- `GET /ndpi/status` — returns current desired state based on config contents.
- `POST /ndpi/enable` — enables the nDPI plugin and restarts Suricata.
- `POST /ndpi/disable` — disables the nDPI plugin and restarts Suricata.

### Example usage

Start host-agent:

- `sudo ./bin/host-agent serve --config config/config.yaml --sock /run/ndpi-agent.sock`

Enable nDPI:

- `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/enable`

Disable nDPI:

- `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/disable`

### Operational notes

Enabling/disabling the plugin is a restart-level change and may briefly interrupt traffic inspection during Suricata restart.
Reload operations (`suricatasc`, ExecReload) are suitable for reloadable changes (rules/config updates) but are not reliable for dynamic plugin (un)loading.

## Host Agent deployment (systemd)

> Assumption: `host-agent` is installed at `/usr/local/bin/host-agent`, and config is at `/etc/integration-suricata-ndpi/config.yaml`.

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

Optional: “always-on” mode (run service even without incoming requests):

```bash
sudo systemctl enable --now ndpi-agent.service
```

Update binary:

```bash
sudo systemctl stop ndpi-agent.service ndpi-agent.socket
sudo install -m 0755 bin/host-agent /usr/local/bin/host-agent
sudo systemctl start ndpi-agent.socket
```

## Host Agent API (Unix socket)

Default socket: `/run/ndpi-agent.sock`

### Liveness

- `sudo curl --unix-socket /run/ndpi-agent.sock http://localhost/health`

### Ensure Suricata control socket is reachable (and restart Suricata if needed)

- `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/suricata/ensure`

### nDPI status / enable / disable

- status: `sudo curl --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/status`
- enable: `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/enable`
- disable: `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/disable`

## Integration service API (TCP)

Default listen: `http.addr` (example `:8080`).

### Health (GET only)

- `curl http://localhost:8080/health`

### Plan/apply

- `curl http://localhost:8080/plan`
- `curl -X POST http://localhost:8080/apply`

### nDPI toggle via integration (delegates to host-agent over Unix socket)

- `curl -X POST http://localhost:8080/ndpi/enable`
- `curl -X POST http://localhost:8080/ndpi/disable`

## Operational commands (system)

### Check service/socket state

```bash
sudo systemctl status ndpi-agent.socket
sudo systemctl status ndpi-agent.service
```

## Requirements

- Suricata installed and managed by systemd (service name: `system.suricata_service`, usually `suricata`).
- Suricata control socket available (one of `suricata.socket_candidates`, example `/run/suricata/suricata-command.socket`).
- Host agent installed: `/usr/local/bin/host-agent`.
- Host agent config present: `/etc/integration-suricata-ndpi/config.yaml`.
- nDPI plugin file exists on host: `paths.ndpi_plugin_path` (example `/usr/local/lib/suricata/ndpi.so`).
- Permission to restart Suricata via systemctl (host-agent runs as root via systemd unit).

## What is enabled / included

Socket activation via `ndpi-agent.socket` (recommended default): host-agent starts on first request to `/run/ndpi-agent.sock`.

Host-agent endpoints:

- `/health` (GET)
- `/suricata/ensure` (POST)
- `/ndpi/status` (GET)
- `/ndpi/enable` (POST)
- `/ndpi/disable` (POST)
