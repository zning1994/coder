package cli

import (
	"github.com/spf13/cobra"
)

func templateVersions() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "versions",
		Aliases: []string{"version"},
	}
	cmd.AddCommand(
		templateVersionList(),
	)
	return cmd
}
