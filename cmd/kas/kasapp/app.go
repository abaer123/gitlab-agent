package kasapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type App struct {
	ConfigurationFile         string
	ListenNetwork             string
	ListenAddress             string
	ListenWebSocket           bool
	GitLabAddress             string
	GitLabAuthSecretFile      string
	ReloadConfigurationPeriod time.Duration
}

func (a *App) Run(ctx context.Context) error {
	cfg, err := a.maybeLoadConfigurationFile()
	if err != nil {
		return err
	}
	ApplyDefaultsToKasConfigurationFile(cfg)
	if a.ListenNetwork != defaultListenNetwork.String() {
		val, ok := kascfg.ListenNetworkEnum_value[a.ListenNetwork]
		if !ok {
			return fmt.Errorf("unsupported listen-network flag passed: %s", a.ListenNetwork)
		}
		cfg.Listen.Network = kascfg.ListenNetworkEnum(val)
	}
	if a.ListenAddress != defaultListenAddress {
		cfg.Listen.Address = a.ListenAddress
	}
	if a.ListenWebSocket {
		cfg.Listen.Websocket = a.ListenWebSocket
	}

	if a.GitLabAddress != defaultGitLabAddress {
		cfg.Gitlab.Address = a.GitLabAddress
	}
	if a.GitLabAuthSecretFile != "" {
		cfg.Gitlab.AuthenticationSecretFile = a.GitLabAuthSecretFile
	}

	if a.ReloadConfigurationPeriod != defaultAgentConfigurationPollPeriod {
		cfg.Agent.Configuration.PollPeriod = durationpb.New(a.ReloadConfigurationPeriod)
	}
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
	flagset.StringVar(&app.ListenNetwork, "listen-network", defaultListenNetwork.String(), "Network type to listen on. Supported values: tcp, tcp4, tcp6, unix")
	flagset.StringVar(&app.ListenAddress, "listen-address", defaultListenAddress, "Address to listen on")
	flagset.BoolVar(&app.ListenWebSocket, "listen-websocket", false, "Enable \"gRPC through WebSocket\" listening mode. Rather than expecting gRPC directly, expect a WebSocket connection, from which a gRPC stream is then unpacked")
	flagset.StringVar(&app.GitLabAddress, "gitlab-address", defaultGitLabAddress, "GitLab address")
	flagset.StringVar(&app.GitLabAuthSecretFile, "authentication-secret-file", "", "File with JWT secret to authenticate with GitLab")
	flagset.DurationVar(&app.ReloadConfigurationPeriod, "reload-configuration-period", defaultAgentConfigurationPollPeriod, "How often to reload agentk configuration")
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
