//go:build wireinject

package wire

import (
	"integration-suricata-ndpi/internal/app"

	"github.com/google/wire"
)

func InitializeIntegrationService(configPath string) (app.Service, error) {
	wire.Build(newIntegrationService)
	return nil, nil
}

func InitializeHostAgentService(opts HostAgentOptions) (app.Service, error) {
	wire.Build(newHostAgentService)
	return nil, nil
}
