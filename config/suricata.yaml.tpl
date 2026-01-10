%YAML 1.1
---

# This configuration file is intentionally trimmed from the Suricata default.
# Keep Suricata section order/style, remove unused commented blocks.

suricata-version: "8.0"

##
## Step 1: Inform Suricata about your network
##

vars:
  address-groups:
    HOME_NET: "[192.168.0.0/16,10.0.0.0/8,172.16.0.0/12]"
    EXTERNAL_NET: "!$HOME_NET"

    HTTP_SERVERS: "$HOME_NET"
    SMTP_SERVERS: "$HOME_NET"
    SQL_SERVERS: "$HOME_NET"
    DNS_SERVERS: "$HOME_NET"
    TELNET_SERVERS: "$HOME_NET"
    AIM_SERVERS: "$EXTERNAL_NET"
    DC_SERVERS: "$HOME_NET"
    DNP3_SERVER: "$HOME_NET"
    DNP3_CLIENT: "$HOME_NET"
    MODBUS_CLIENT: "$HOME_NET"
    MODBUS_SERVER: "$HOME_NET"
    ENIP_CLIENT: "$HOME_NET"
    ENIP_SERVER: "$HOME_NET"

  port-groups:
    HTTP_PORTS: "80"
    SHELLCODE_PORTS: "!80"
    ORACLE_PORTS: 1521
    SSH_PORTS: "22"
    DNP3_PORTS: "20000"
    MODBUS_PORTS: "502"
    FILE_DATA_PORTS: "[$HTTP_PORTS,110,143]"
    FTP_PORTS: "21"
    GENEVE_PORTS: "6081"
    VXLAN_PORTS: "4789"
    TEREDO_PORTS: "3544"
    SIP_PORTS: "[5060, 5061]"

##
## Step 2: Select outputs to enable
##

default-log-dir: /var/log/suricata/

stats:
  enabled: yes
  interval: 8

# Plugins (nDPI)
plugins:
  - /usr/local/lib/suricata/ndpi.so
  # альтернативно:
  # - /usr/lib/suricata/ndpi.so

outputs:
  - fast:
      enabled: yes
      filename: fast.log
      append: yes

  - eve-log:
      enabled: yes
      filetype: regular
      filename: eve.json

      pcap-file: false
      community-id: false
      community-id-seed: 0

      types:
        - alert:
            tagged-packets: yes
        - flow
        - http:
            extended: yes
        - dns:
        - tls:
            extended: yes
        - files:
        - stats:
            totals: yes
            threads: no

  - stats:
      enabled: yes
      filename: stats.log
      append: yes
      totals: yes
      threads: no

##
## Logging about what Suricata is doing (engine log)
##

logging:
  default-log-level: notice
  outputs:
    - console:
        enabled: yes
    - file:
        enabled: yes
        level: info
        filename: suricata.log

##
## Step 3: Configure common capture settings
##

af-packet:
  - interface: enp0s3
    cluster-id: 99
    cluster-type: cluster_flow
    defrag: yes

  - interface: default

##
## Step 4: App Layer Protocol configuration
## 
##

app-layer:
  protocols:
    http:
      enabled: yes
      libhtp:
        default-config:
          personality: IDS
          request-body-limit: 100 KiB
          response-body-limit: 100 KiB

    tls:
      enabled: yes
      detection-ports:
        dp: 443

    dns:
      tcp:
        enabled: yes
        detection-ports:
          dp: 53
      udp:
        enabled: yes
        detection-ports:
          dp: 53

    ssh:
      enabled: yes
    smtp:
      enabled: yes
    ftp:
      enabled: yes
    rdp:
      enabled: yes
    smb:
      enabled: yes
      detection-ports:
        dp: 139, 445
    mqtt:
      enabled: yes
    http2:
      enabled: yes
    quic:
      enabled: yes
    ike:
      enabled: yes
    snmp:
      enabled: yes
    ldap:
      tcp:
        enabled: yes
        detection-ports:
          dp: 389, 3268
      udp:
        enabled: yes
        detection-ports:
          dp: 389, 3268
    dhcp:
      enabled: yes
    mdns:
      enabled: yes
    ntp:
      enabled: yes

##
## Run Options / unix socket (IMPORTANT for suricatasc + integration)
##

unix-command:
  enabled: yes
  filename: /run/suricata/suricata-command.socket
  mode: 0660

legacy:
  uricontent: enabled

##
## Detection / performance (разумный минимум)
##

exception-policy: auto

pcre:
  match-limit: 3500
  match-limit-recursion: 1500

stream:
  memcap: 64 MiB
  checksum-validation: yes
  inline: auto
  reassembly:
    memcap: 256 MiB
    depth: 1 MiB

detect:
  profile: medium
  prefilter:
    default: mpm

mpm-algo: auto
spm-algo: auto

##
## Configure Suricata-Update managed rules
##

default-rule-path: /var/lib/suricata/rules

rule-files:
  - suricata.rules
  - ndpi/*.rules

##
## Auxiliary configuration files.
##

classification-file: /etc/suricata/classification.config
reference-config-file: /etc/suricata/reference.config
