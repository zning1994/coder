package monitoring_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/databasefake"
	"github.com/coder/coder/coderd/monitoring"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
	stats := monitoring.New()
	registry := prometheus.NewRegistry()
	registry.MustRegister(stats.UsersTotal, stats.Workspaces, stats.WorkspaceResources)

	ctx := context.Background()
	db := databasefake.New()
	user, _ := db.InsertUser(ctx, database.InsertUserParams{
		ID:       uuid.New(),
		Username: "kyle",
	})
	org, _ := db.InsertOrganization(ctx, database.InsertOrganizationParams{
		ID:   uuid.New(),
		Name: "potato",
	})
	template, _ := db.InsertTemplate(ctx, database.InsertTemplateParams{
		ID:             uuid.New(),
		Name:           "something",
		OrganizationID: org.ID,
	})
	workspace, _ := db.InsertWorkspace(ctx, database.InsertWorkspaceParams{
		ID:             uuid.New(),
		OwnerID:        user.ID,
		OrganizationID: org.ID,
		TemplateID:     template.ID,
		Name:           "banana",
	})
	job, _ := db.InsertProvisionerJob(ctx, database.InsertProvisionerJobParams{
		ID:             uuid.New(),
		OrganizationID: org.ID,
	})
	version, _ := db.InsertTemplateVersion(ctx, database.InsertTemplateVersionParams{
		ID: uuid.New(),
		TemplateID: uuid.NullUUID{
			UUID:  template.ID,
			Valid: true,
		},
		CreatedAt:      database.Now(),
		OrganizationID: org.ID,
		JobID:          job.ID,
	})
	db.InsertWorkspaceBuild(ctx, database.InsertWorkspaceBuildParams{
		ID:                uuid.New(),
		JobID:             job.ID,
		WorkspaceID:       workspace.ID,
		TemplateVersionID: version.ID,
		Transition:        database.WorkspaceTransitionStart,
	})
	db.InsertWorkspaceResource(ctx, database.InsertWorkspaceResourceParams{
		ID:    uuid.New(),
		JobID: job.ID,
		Type:  "google_compute_instance",
		Name:  "banana",
	})
	db.InsertWorkspaceResource(ctx, database.InsertWorkspaceResourceParams{
		ID:    uuid.New(),
		JobID: job.ID,
		Type:  "google_compute_instance",
		Name:  "banana",
	})
	db.InsertWorkspace(ctx, database.InsertWorkspaceParams{
		ID:             uuid.New(),
		OwnerID:        user.ID,
		OrganizationID: org.ID,
		TemplateID:     template.ID,
		Name:           "banana2",
	})

	err := stats.Refresh(ctx, db)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	for _, metric := range metrics {

		if *metric.Name == "coder_resources" {
			// fmt.Printf("%+v\n", metric)
			for _, m := range metric.Metric {
				fmt.Printf("START: %+v\n\n\n", m)
			}
		}

	}

	// fmt.Printf("Metrics: %+v\n", metrics)

	// t.Parallel()
	// t.Run("Example", func(t *testing.T) {
	// })
}
