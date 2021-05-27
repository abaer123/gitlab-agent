package kasapp

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type App struct {
	ConfigurationFile string
}

func (a *App) Run(ctx context.Context) (retErr error) {
	cfg, err := LoadConfigurationFile(a.ConfigurationFile)
	if err != nil {
		return err
	}
	ApplyDefaultsToKasConfigurationFile(cfg)
	err = cfg.ValidateExtra()
	if err != nil {
		return fmt.Errorf("kascfg.ValidateExtra: %v", err)
	}
	logger, err := loggerFromConfig(cfg.Observability.Logging)
	if err != nil {
		return err
	}
	defer errz.SafeCall(logger.Sync, &retErr)
	// Kubernetes uses klog so here we pipe all logs from it to our logger via an adapter.
	klog.SetLogger(zapr.NewLogger(logger))
	app := ConfiguredApp{
		Log:           logger,
		Configuration: cfg,
	}
	return app.Run(ctx)
}

func LoadConfigurationFile(configFile string) (*kascfg.ConfigurationFile, error) {
	configYAML, err := ioutil.ReadFile(configFile) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("configuration file: %v", err)
	}
	configJSON, err := yaml.YAMLToJSON(configYAML)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %v", err)
	}
	cfg := &kascfg.ConfigurationFile{}
	err = protojson.Unmarshal(configJSON, cfg)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %v", err)
	}
	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("kascfg.Validate: %v", err)
	}
	return cfg, nil
}

func NewFromFlags(flagset *pflag.FlagSet, programName string, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.StringVar(&app.ConfigurationFile, "configuration-file", "", "Optional configuration file to use (YAML)")
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}

func loggerFromConfig(loggingCfg *kascfg.LoggingCF) (*zap.Logger, error) {
	level, err := logz.LevelFromString(loggingCfg.Level.String())
	if err != nil {
		return nil, err
	}
	return logz.LoggerWithLevel(level), nil
}
