package generate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
)

const (
	kustomizationPathEnvVar     = "KPT_PACKAGE_PATH"
	kustomizationAgentTokenPath = "base/secrets/agent.token"
	kustomizationBaseOverlay    = "base"
	kustomizationRbacOverlay    = "cluster"
	warningText                 = `
###
# WARNING: output contains the agent token, which should be considered sensitive and never committed to source control
###

`  // should end with two newline characters
)

type GenerateCmd struct {
	KustomizationPath string
	AgentToken        string
	AgentVersion      string
	KasAddress        string
	Namespace         string
	NoRbac            bool
}

func NewFromFlags(flagset *pflag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &GenerateCmd{}
	app.KustomizationPath = os.Getenv(kustomizationPathEnvVar)

	flagset.StringVar(&app.AgentToken, "agent-token", "", "Access token registered for agent")
	flagset.StringVar(&app.AgentVersion, "agent-version", cmd.Version, "Version of the agentk image to use")
	flagset.StringVar(&app.KasAddress, "kas-address", "", "GitLab Kubernetes Agent Server address")
	flagset.StringVar(&app.Namespace, "namespace", "gitlab-agent", "Kubernetes namespace to create resources in")
	flagset.BoolVar(&app.NoRbac, "no-rbac", false, "Do not include corresponding Roles and RoleBindings for the agent service account")

	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}

func (c *GenerateCmd) Run(ctx context.Context) (retErr error) {
	overlay := kustomizationRbacOverlay

	if c.NoRbac {
		overlay = kustomizationBaseOverlay
	}

	if err := c.writeTokenFile(); err != nil {
		return err
	}
	if err := c.kustomizeSet(ctx, "agent-version", c.AgentVersion); err != nil {
		return err
	}
	if err := c.kustomizeSet(ctx, "kas-address", c.KasAddress); err != nil {
		return err
	}
	if err := c.kustomizeSet(ctx, "namespace", c.Namespace); err != nil {
		return err
	}

	if err := c.kustomizeBuild(ctx, overlay); err != nil {
		return err
	}

	return nil
}

func (c *GenerateCmd) kustomizeSet(ctx context.Context, setKey, value string) error {
	if value == "" {
		return fmt.Errorf("--%v is required", setKey)
	}
	cmdctx := exec.CommandContext(ctx, "kustomize", "cfg", "set", c.KustomizationPath, setKey, value) // nolint:gosec

	// Ignoring stdout, piping only stderr
	cmdctx.Stderr = os.Stderr

	return cmdctx.Run() //nolint:gosec
}

func (c *GenerateCmd) kustomizeBuild(ctx context.Context, overlay string) error {
	buildPath := filepath.Join(c.KustomizationPath, overlay)
	cmdctx := exec.CommandContext(ctx, "kustomize", "build", buildPath) //nolint:gosec

	var out bytes.Buffer
	out.WriteString(warningText)
	cmdctx.Stdout = &out
	cmdctx.Stderr = os.Stderr

	if err := cmdctx.Run(); err != nil {
		return err
	}
	_, _ = os.Stdout.Write(out.Bytes())
	return nil
}

func (c *GenerateCmd) writeTokenFile() error {
	if c.AgentToken == "" {
		return errors.New("--agent-token is required")
	}

	tokenFilePath := filepath.Join(c.KustomizationPath, kustomizationAgentTokenPath)
	return ioutil.WriteFile(tokenFilePath, []byte(c.AgentToken), 0777) //nolint:gosec
}
