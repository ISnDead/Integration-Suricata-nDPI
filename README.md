### Integration Suricata + nDPI automates configuration management for Suricata deployments that use the nDPI plugin.**

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
  - paths.ndpi_rules_local — local nDPI rules directory
  - paths.suricata_template — suricata.yaml.tpl template
  - paths.suricatasc — path to suricatasc
  - suricata.socket_candidates — unix-socket path candidates
  - suricata.config_candidates — suricata.yaml path candidates
  - reload.command, reload.timeout — best-effort reload/reconfigure parameters
