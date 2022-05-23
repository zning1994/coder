package cli

import (
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/codersdk"
)

func cancelBuild() *cobra.Command {
	return &cobra.Command{
		Annotations: workspaceCommand,
		Use:         "cancel-build <workspace>",
		Short:       "Cancels the currently running build for the specified workspace.",
		Aliases:     []string{"cb"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := createClient(cmd)
			if err != nil {
				return err
			}
			organization, err := currentOrganization(cmd, client)
			if err != nil {
				return err
			}

			workspace, err := client.WorkspaceByOwnerAndName(ctx, organization.ID, codersdk.Me, args[0])
			if err != nil {
				return xerrors.Errorf("workspace by owner and name: %w", err)
			}

			err = client.CancelWorkspaceBuild(ctx, workspace.LatestBuild.ID)
			if err != nil {
				return xerrors.Errorf("cancel workspace build: %w", err)
			}

			return nil
		},
	}
}
