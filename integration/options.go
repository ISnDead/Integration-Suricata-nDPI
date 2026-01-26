package integration

import (
	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
)

func buildRunnerOptions(cfg *config.Config, runner executil.Runner, fs fsutil.FS) RunnerOptions {
	paths := cfg.Paths
	suricata := cfg.Suricata
	reload := cfg.Reload
	ndpi := cfg.NDPI

	return RunnerOptions{
		Apply: ApplyConfigOptions{
			TemplatePath:     paths.SuricataTemplate,
			ConfigCandidates: suricata.ConfigCandidates,
			SocketCandidates: suricata.SocketCandidates,
			SuricataSCPath:   paths.SuricataSC,
			ReloadCommand:    reload.Command,
			ReloadTimeout:    reload.Timeout,
			CommandRunner:    runner,
			FS:               fs,
		},
		NDPIValidate: NDPIValidateOptions{
			NDPIPluginPath:       paths.NDPIPluginPath,
			NDPIRulesDir:         paths.NDPIRulesLocal,
			SuricataTemplatePath: paths.SuricataTemplate,
			SuricataSCPath:       paths.SuricataSC,
			ReloadCommand:        reload.Command,
			ReloadTimeout:        reload.Timeout,
			ExpectedRulesPattern: ndpi.ExpectedRulesPattern,
			FS:                   fs,
		},
		SuricataStart: SuricataStartOptions{
			SocketCandidates: suricata.SocketCandidates,
			SystemctlPath:    cfg.System.Systemctl,
			SystemdUnit:      cfg.System.SuricataService,
			StartTimeout:     suricata.StartTimeout,
		},
	}
}
