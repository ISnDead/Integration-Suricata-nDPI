# Instructions for developing rules 
A rule/signature consists of the following:
- the action, determining what happens when the rule matches.
- the header, defining the protocol, IP addresses, ports and direction of the rule.
- the rule options, defining the specifics of the rule.

An example of a rule is as follows:\
***alert http $HOME_NET any -> $EXTERNAL_NET any (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)***

In this example, "**alert**" is the action, "**http $HOME_NET any -> $EXTERNAL_NET any**" is the header and **other** are the options.
## Action
***alert** http $HOME_NET any -> $EXTERNAL_NET any (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)*

Valid actions are:
- alert — generate an alert.
- pass — stop further inspection of the packet.
- drop — drop packet and generate alert.
- reject — send RST/ICMP unreach error to the sender of the matching packet.
- rejectsrc — same as just reject.
- rejectdst — send RST/ICMP error packet to receiver of the matching packet.
- rejectboth — send RST/ICMP error packets to both sides of the conversation.
## Protocol
*alert **http** $HOME_NET any -> $EXTERNAL_NET any (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)*

This keyword in a signature tells Suricata which protocol it concerns. You can choose between four basic protocols:
- tcp (for tcp-traffic)
- udp
- icmp
- ip (ip stands for 'all' or 'any')\
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
- websocket\
The availability of these protocols depends on whether the protocol is enabled in the configuration file, suricata.yaml.
If you have a signature with the protocol declared as 'http', Suricata makes sure the signature will only match if the TCP stream contains http traffic.
## Source and destination
*alert http **$HOME_NET** any -> **$EXTERNAL_NET** any (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)*

The first emphasized part is the traffic source, the second is the traffic destination (note the direction of the directional arrow).
With the source and destination, you specify the source of the traffic and the destination of the traffic, respectively. You can assign IP addresses, (both IPv4 and IPv6 are supported) and IP ranges. These can be combined with operators:
- ../.. — IP ranges (CIDR notation)
- ! — exception/negation
- [..,..] — grouping\
Normally, you would also make use of variables, such as $HOME_NET and $EXTERNAL_NET. The suricata.yaml configuration file specifies the IP addresses these concern. The respective $HOME_NET and $EXTERNAL_NET settings will be used in place of the variables in your rules.
## Ports (source and destination)
*alert http $HOME_NET **any** -> $EXTERNAL_NET **any** (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)*

The first emphasized part is the source port, the second is the destination port (note the direction of the directional arrow).
Traffic comes in and goes out through ports. Different protocols have different port numbers. For example, the default port for HTTP is 80 while 443 is typically the port for HTTPS. Note, however, that the port does not dictate which protocol is used in the communication. Rather, it determines which application is receiving the data.
In setting ports you can make use of special operators as well. Operators such as:
- : — port ranges
- ! — exception/negation
- [.., ..] — grouping\
## Direction
*alert http $HOME_NET any **->** $EXTERNAL_NET any (msg:"HTTP GET Request Containing Rule in URI"; flow:established,to_server; http.method; content:"GET"; http.uri; content:"rule"; fast_pattern; classtype:bad-unknown; sid:123; rev:1;)*

The directional arrow indicates which way the signature will be evaluated. In most signatures an arrow to the right (->) is used. This means that only packets with the same direction can match. There is also the double arrow (=>), which respects the directionality as ->, but allows matching on bidirectional transactions, used with keywords matching each direction. Finally, it is also possible to have a rule match either directions (<>).
## Rule options
The rest of the rule consists of options. These are enclosed by parenthesis and separated by semicolons. Some options have settings (such as msg), which are specified by the keyword of the option, followed by a colon, followed by the settings. Others have no settings; they are simply the keyword (such as nocase).

Rule options have a specific ordering and changing their order would change the meaning of the rule.\
The characters **;** and **"** have special meaning in the Suricata rule language and must be escaped when used in a rule option value. For example:\
**msg:"Message with semicolon\;";**\
As a consequence, you must also escape the backslash, as it functions as an escape character.\
