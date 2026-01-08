package integration

import "net"

type SuricataClient struct {
	Conn net.Conn
	Path string
}

const (
	// SocketPath берем из: /var/run/suricata/suricata-command.socket
	SocketPath = "/var/run/suricata/suricata-command.socket"

	// Путь к папке, где лежат файлы правил nDPI
	NDPIRulesLocalPath = "rules/ndpi/"

	// Путь к шаблону конфигурации Suricata в проекте
	SuricataTemplatePath = "config/suricata.yaml.tpl"

	// Системный путь к рабочему конфигу Suricata
	SuricataConfigPath = "/etc/suricata/suricata.yaml"
)
