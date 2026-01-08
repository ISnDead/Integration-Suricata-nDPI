package integration

import "net"

type SuricataClient struct {
	Conn net.Conn
	Path string
}

const (
	// Путь к папке, где лежат файлы правил nDPI
	NDPIRulesLocalPath = "rules/ndpi/"

	// Путь к шаблону конфигурации Suricata в проекте
	SuricataTemplatePath = "config/suricata.yaml.tpl"
)

var (
	SuricataSocketCandidates = []string{
		"/var/run/suricata/suricata-command.socket",
		"/usr/local/var/run/suricata/suricata-command.socket",
	}

	SuricataConfigCandidates = []string{
		"/etc/suricata/suricata.yaml",
		"/usr/local/etc/suricata/suricata.yaml",
	}
)
