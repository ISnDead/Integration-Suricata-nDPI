%YAML 1.1
---

suricata-version: "8.0"

vars:
  address-groups:
    HOME_NET: "[192.168.0.0/16,10.0.0.0/8,172.16.0.0/12]"
    EXTERNAL_NET: "!$HOME_NET"
  port-groups:
    HTTP_PORTS: "80"
    HTTPS_PORTS: "443"
    DNS_PORTS: "53"

default-log-dir: /var/log/suricata/

# nDPI plugin
plugins:
  # оставь тот путь, который реально существует в твоём образе/хосте:
  # - /usr/lib/suricata/ndpi.so
  - /usr/local/lib/suricata/ndpi.so

outputs:
  - fast:
      enabled: yes
      filename: fast.log
      append: yes

  - eve-log:
      enabled: yes
      filetype: regular
      filename: eve.json
      types:
        - alert:
            tagged-packets: yes
        - flow
        - stats:
            totals: yes
            threads: no

# захват трафика (минимально; подставь свой интерфейс)
af-packet:
  - interface: enp0s3
  - interface: default

default-rule-path: /var/lib/suricata/rules
rule-files:
  - suricata.rules

classification-file: /etc/suricata/classification.config
reference-config-file: /etc/suricata/reference.config
