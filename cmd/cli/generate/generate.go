package generate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
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
	StdOut, StdErr    io.Writer
}

func NewCommand() *cobra.Command {
	a := GenerateCmd{}
	a.KustomizationPath = os.Getenv(kustomizationPathEnvVar)
	c := &cobra.Command{
		Use:   "generate",
		Short: "Prints the YAML manifests based on specified configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.StdOut = cmd.OutOrStdout()
			a.StdErr = cmd.ErrOrStderr()
			return a.Run(cmd.Context())
		},
	}
	f := c.Flags()
	f.StringVar(&a.AgentToken, "agent-token", "", "Access token registered for agent")
	f.StringVar(&a.AgentVersion, "agent-version", cmd.Version, "Version of the agentk image to use")
	f.StringVar(&a.KasAddress, "kas-address", "", "GitLab Kubernetes Agent Server address")
	f.StringVar(&a.Namespace, "namespace", "gitlab-agent", "Kubernetes namespace to create resources in")
	f.BoolVar(&a.NoRbac, "no-rbac", false, "Do not include corresponding Roles and RoleBindings for the agent service account")
	cobra.CheckErr(c.MarkFlagRequired("agent-token"))
	return c
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
	cmdctx.Stderr = c.StdErr

	return cmdctx.Run()
}

func (c *GenerateCmd) kustomizeBuild(ctx context.Context, overlay string) error {
	buildPath := filepath.Join(c.KustomizationPath, overlay)
	cmdctx := exec.CommandContext(ctx, "kustomize", "build", buildPath) //nolint:gosec

	var out bytes.Buffer
	out.WriteString(warningText)
	cmdctx.Stdout = &out
	cmdctx.Stderr = c.StdErr

	if err := cmdctx.Run(); err != nil {
		return err
	}
	_, err := c.StdOut.Write(out.Bytes())
	return err
}

func (c *GenerateCmd) writeTokenFile() error {
	tokenFilePath := filepath.Join(c.KustomizationPath, kustomizationAgentTokenPath)
	return os.WriteFile(tokenFilePath, []byte(c.AgentToken), 0777) //nolint:gosec
}
