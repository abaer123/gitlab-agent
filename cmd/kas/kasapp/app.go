package kasapp

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
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
		return fmt.Errorf("kascfg.ValidateExtra: %w", err)
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
	configYAML, err := os.ReadFile(configFile) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("configuration file: %w", err)
	}
	configJSON, err := yaml.YAMLToJSON(configYAML)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %w", err)
	}
	cfg := &kascfg.ConfigurationFile{}
	err = protojson.Unmarshal(configJSON, cfg)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %w", err)
	}
	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("kascfg.Validate: %w", err)
	}
	return cfg, nil
}

func NewCommand() *cobra.Command {
	a := App{}
	c := &cobra.Command{
		Use:   "kas",
		Short: "GitLab Kubernetes Agent Server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run(cmd.Context())
		},
	}
	c.Flags().StringVar(&a.ConfigurationFile, "configuration-file", "", "Configuration file to use (YAML)")
	cobra.CheckErr(c.MarkFlagRequired("configuration-file"))

	return c
}

func loggerFromConfig(loggingCfg *kascfg.LoggingCF) (*zap.Logger, error) {
	level, err := logz.LevelFromString(loggingCfg.Level.String())
	if err != nil {
		return nil, err
	}
	return logz.LoggerWithLevel(level), nil
}
