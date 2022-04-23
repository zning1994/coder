package cliui

import (
	"fmt"
	"io"

	"github.com/coder/coder/codersdk"
	"github.com/jedib0t/go-pretty/v6/table"
)

func TemplateVersions(writer io.Writer, template codersdk.Template, versions []codersdk.TemplateVersion) error {
	tableWriter := table.NewWriter()
	tableWriter.SetStyle(table.StyleLight)
	tableWriter.Style().Options.SeparateColumns = false
	tableWriter.AppendHeader(table.Row{"Name", "Status", "Duration"})
	for _, version := range versions {
		var status string
		switch version.Job.Status {
		case codersdk.ProvisionerJobCanceled:
			status = "Canceled"
		case codersdk.ProvisionerJobCanceling:
			status = "Canceling..."
		case codersdk.ProvisionerJobPending:
			status = Styles.Placeholder.Render("Queued")
		case codersdk.ProvisionerJobRunning:
			status = "Building"
		case codersdk.ProvisionerJobFailed:
			status = "Failed"
		case codersdk.ProvisionerJobSucceeded:
			status = "Succeeded"
		}

		name := version.Name
		if version.ID == template.ActiveVersionID {
			name = Styles.Bold.Render(name)
		}
		tableWriter.AppendRow(table.Row{
			name,
			status,
			"Something",
		})
	}
	_, err := fmt.Fprintln(writer, tableWriter.Render())
	return err
}
