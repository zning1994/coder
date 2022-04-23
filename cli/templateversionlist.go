package cli

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func templateVersionList() *cobra.Command {
	return &cobra.Command{
		Use:     "list <template>",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}

			organization, err := currentOrganization(cmd, client)
			if err != nil {
				return err
			}

			template, err := client.TemplateByName(cmd.Context(), organization.ID, args[0])
			if err != nil {
				return err
			}

			versions, err := client.TemplateVersionsByTemplate(cmd.Context(), template.ID)
			if err != nil {
				return err
			}

			tableWriter := table.NewWriter()
			tableWriter.SetStyle(table.StyleLight)
			tableWriter.Style().Options.SeparateColumns = false
			tableWriter.AppendHeader(table.Row{"Name", "Status", "Duration"})
			for _, version := range versions {
				tableWriter.AppendRow(table.Row{
					"👑 " + version.Name,
					version.Job.Status,
					"Something",
				})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), tableWriter.Render())
			return err
		},
	}
}
