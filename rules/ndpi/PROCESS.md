# Instructions for developing rules
A rule/signature consists of the following:
- the action, determining what happens when the rule matches.
- the header, defining the protocol, IP addresses, ports and direction of the rule.
- the rule options, defining the specifics of the rule.
An example of a rule is as follows:
alert http $HOME_NET any -> $EXTERNAL_NET any (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)
In this example, red is the action, green is the header and blue are the options.
## Action
Valid actions are:
- alert - generate an alert.
- pass - stop further inspection of the packet.
- drop - drop packet and generate alert.
- reject - send RST/ICMP unreach error to the sender of the matching packet.
- rejectsrc - same as just reject.
- rejectdst - send RST/ICMP error packet to receiver of the matching packet.
- rejectboth - send RST/ICMP error packets to both sides of the conversation.
## Protocol
This keyword in a signature tells Suricata which protocol it concerns. You can choose between four basic protocols:
- tcp (for tcp-traffic)
- udp
- icmp
- ip (ip stands for 'all' or 'any')
There are a couple of additional TCP related protocol options:
- tcp-pkt (for matching content in individual tcp packets)
- tcp-stream (for matching content only in a reassembled tcp stream)
There are also a few so-called application layer protocols, or layer 7 protocols you can pick from. These are:
- http (either HTTP1 or HTTP2)
- http1
- http2
- ftp
- tls (this includes ssl)
- smb
- dns
- dcerpc
- dhcp
- ssh
- smtp
- imap
- pop3
- modbus (disabled by default)
- dnp3 (disabled by default)
- enip (disabled by default)
- nfs
- ike
- krb5
- bittorrent-dht
- ntp
- dhcp
- rfb
- rdp
- snmp
- tftp
- sip
- websocket
The availability of these protocols depends on whether the protocol is enabled in the configuration file, suricata.yaml.
If you have a signature with the protocol declared as 'http', Suricata makes sure the signature will only match if the TCP stream contains http traffic.
