package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/codersdk"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func vscode() *cobra.Command {
	return &cobra.Command{
		Use:  "vscode <workspace>",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}

			workspaceParts := strings.Split(args[0], ".")
			workspace, err := client.WorkspaceByName(cmd.Context(), codersdk.Me, workspaceParts[0])
			if err != nil {
				return err
			}

			if workspace.LatestBuild.Transition != database.WorkspaceTransitionStart {
				return xerrors.New("workspace must be in start transition to ssh")
			}

			if workspace.LatestBuild.Job.CompletedAt == nil {
				err = cliui.WorkspaceBuild(cmd.Context(), cmd.ErrOrStderr(), client, workspace.LatestBuild.ID, workspace.CreatedAt)
				if err != nil {
					return err
				}
			}

			if workspace.LatestBuild.Transition == database.WorkspaceTransitionDelete {
				return xerrors.New("workspace is deleting...")
			}

			resources, err := client.WorkspaceResourcesByBuild(cmd.Context(), workspace.LatestBuild.ID)
			if err != nil {
				return err
			}

			agents := make([]codersdk.WorkspaceAgent, 0)
			for _, resource := range resources {
				agents = append(agents, resource.Agents...)
			}
			if len(agents) == 0 {
				return xerrors.New("workspace has no agents")
			}
			var agent codersdk.WorkspaceAgent
			if len(workspaceParts) >= 2 {
				for _, otherAgent := range agents {
					if otherAgent.Name != workspaceParts[1] {
						continue
					}
					agent = otherAgent
					break
				}
				if agent.ID == uuid.Nil {
					return xerrors.Errorf("agent not found by name %q", workspaceParts[1])
				}
			}
			if agent.ID == uuid.Nil {
				if len(agents) > 1 {
					return xerrors.New("you must specify the name of an agent")
				}
				agent = agents[0]
			}
			// OpenSSH passes stderr directly to the calling TTY.
			// This is required in "stdio" mode so a connecting indicator can be displayed.
			err = cliui.Agent(cmd.Context(), cmd.ErrOrStderr(), cliui.AgentOptions{
				WorkspaceName: workspace.Name,
				Fetch: func(ctx context.Context) (codersdk.WorkspaceAgent, error) {
					return client.WorkspaceAgent(ctx, agent.ID)
				},
			})
			if err != nil {
				return xerrors.Errorf("await agent: %w", err)
			}

			vscodePath, err := exec.LookPath("code")
			if err != nil {
				return xerrors.New(`The "code" binary must exist on your path!`)
			}
			output, err := exec.CommandContext(cmd.Context(), vscodePath, "--list-extensions").Output()
			if err != nil {
				return xerrors.Errorf("list extensions: %w", err)
			}
			extensions := strings.Split(string(output), "\n")
			hasRemote := false
			for _, extension := range extensions {
				if extension != "ms-vscode-remote.remote-ssh" {
					continue
				}
				hasRemote = true
				break
			}
			if !hasRemote {
				_, err := cliui.Prompt(cmd, cliui.PromptOptions{
					Text:      "Would you like to install the VS Code Remote SSH extension?",
					IsConfirm: true,
				})
				if err != nil {
					return err
				}
				output, err := exec.CommandContext(cmd.Context(), vscodePath, "--install-extension", "ms-vscode-remote.remote-ssh").CombinedOutput()
				if err != nil {
					return xerrors.Errorf("install remote extension: %w: %s", err, output)
				}
			}
			err = configSSH().RunE(cmd, []string{})
			if err != nil {
				return xerrors.Errorf("config ssh: %w", err)
			}
			output, err = exec.CommandContext(cmd.Context(), vscodePath,
				"--remote", fmt.Sprintf("ssh-remote+coder.%s", args[0])).CombinedOutput()
			if err != nil {
				return xerrors.Errorf("launch vs code remote: %w: %s", err, output)
			}
			return nil
		},
	}
}
