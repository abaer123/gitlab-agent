package kasapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
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
	if a.ListenNetwork != defaultListenNetwork {
		cfg.Listen.Network = a.ListenNetwork
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
	options := Options{
		Configuration: cfg,
	}
	return options.Run(ctx)
}

func (a *App) maybeLoadConfigurationFile() (*kascfg.ConfigurationFile, error) {
	cfg := &kascfg.ConfigurationFile{}
	if a.ConfigurationFile == "" {
		return cfg, nil
	}
	configYAML, err := ioutil.ReadFile(a.ConfigurationFile)
	if err != nil {
		return nil, fmt.Errorf("configuration file: %v", err)
	}
	configJSON, err := yaml.YAMLToJSON(configYAML)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %v", err)
	}
	err = protojson.Unmarshal(configJSON, cfg)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %v", err)
	}
	return cfg, nil
}

func NewFromFlags(flagset *pflag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.StringVar(&app.ConfigurationFile, "configuration-file", "", "Optional configuration file to use (YAML)")
	flagset.StringVar(&app.ListenNetwork, "listen-network", defaultListenNetwork, "Network type to listen on. Supported values: tcp, tcp4, tcp6, unix")
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
