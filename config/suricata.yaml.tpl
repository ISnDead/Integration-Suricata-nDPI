plugins:
  - /usr/local/lib/suricata/ndpi.so

unix-command:
  enabled: yes
  filename: /run/suricata/suricata-command.socket
  mode: 0660