package kasapp

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type App struct {
	ConfigurationFile string
}

func (a *App) Run(ctx context.Context) error {
	cfg, err := a.maybeLoadConfigurationFile()
	if err != nil {
		return err
	}
	ApplyDefaultsToKasConfigurationFile(cfg)
	logger, err := loggerFromConfig(cfg.Observability.Logging)
	if err != nil {
		return err
	}
	defer logger.Sync() // nolint: errcheck
	// Kubernetes uses klog so here we pipe all logs from it to our logger via an adapter.
	klog.SetLogger(zapr.NewLogger(logger))
	app := ConfiguredApp{
		Configuration: cfg,
		Log:           logger,
	}
	return app.Run(ctx)
}

func (a *App) maybeLoadConfigurationFile() (*kascfg.ConfigurationFile, error) {
	if a.ConfigurationFile == "" {
		return &kascfg.ConfigurationFile{}, nil
	}
	return LoadConfigurationFile(a.ConfigurationFile)
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

func NewFromFlags(flagset *pflag.FlagSet, arguments []string) (cmd.Runnable, error) {
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
