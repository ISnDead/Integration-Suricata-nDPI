 Integration Suricata + nDPI automates configuration management for Suricata deployments that use the nDPI plugin.

### Key capabilities:

  - Generates suricata.yaml from a template and writes it atomically (safe update).
  - Performs best-effort reload/reconfigure via suricatasc to apply rule/config changes that Suricata can reload without restarting.
  - Validates local resources and nDPI-related paths (ndpi.so, rules directory, template, suricatasc).
  - Checks Suricata availability via the unix control socket.

### Important note about nDPI plugin activation

  - nDPI plugin activation requires a Suricata process restart.
  - This service does not attempt to restart Suricata. Instead, it focuses on safe configuration updates and reloadable changes (e.g., rules) without downtime.


## Quick Start

### Build
  - `go build -o bin/integration ./cmd/integration`

### Run
  - `sudo ./bin/integration -config config/config.yaml`


### Configuration

  - The configuration is defined by a YAML file (example: config/config.yaml).
  - The Suricata template is stored in config/suricata.yaml.tpl.

### Minimally important fields:
  - paths.ndpi_rules_local - local nDPI rules directory
  - paths.suricata_template - suricata.yaml.tpl template
  - paths.suricatasc - path to suricatasc
  - suricata.socket_candidates - unix-socket path candidates
  - suricata.config_candidates - suricata.yaml path candidates
  - reload.command, reload.timeout - best-effort reload/reconfigure parameters

## nDPI plugin enable/disable (Host Agent)

Because Suricata must be restarted to (un)load the nDPI plugin shared object (ndpi.so), the integration service does not perform plugin toggling directly. Instead, plugin enable/disable is delegated to a Host Agent running on the Suricata host.

### What the Host Agent does

  - Modifies the active Suricata configuration (suricata.yaml) by commenting / uncommenting the ndpi.so plugin line within the plugins: section.
  - Writes the updated configuration atomically to avoid partial writes and corrupted configs.
  - Restarts Suricata via systemd to apply the change reliably.

### Why a Host Agent is required

In typical deployments, the integration service runs in a container, while Suricata runs on the host. Restarting Suricata and editing /etc/suricata/suricata.yaml requires host-level privileges and access to systemd, which is intentionally not granted to the containerized integration service.

### API overview (Unix socket)

The Host Agent exposes a small HTTP API over a Unix socket:

  - GET /health - liveness probe
  - GET /ndpi/status - returns current desired state based on config contents
  - POST /ndpi/enable - enables the nDPI plugin and restarts Suricata
  - POST /ndpi/disable - disables the nDPI plugin and restarts Suricata

### Example usage

Enable nDPI
  - `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/enable`

Disable nDPI
  - `sudo curl -X POST --unix-socket /run/ndpi-agent.sock http://localhost/ndpi/disable`

### Operational notes

Enabling/disabling the plugin is a restart-level change and may briefly interrupt traffic inspection during Suricata restart.
Reload operations (suricatasc, ExecReload) are suitable for reloadable changes (rules/config updates) but are not reliable for dynamic plugin (un)loading.
