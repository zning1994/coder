package monitoring

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/database"
)

func New() Stats {
	stats := Stats{
		AgentConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "coder",
			Name:      "agent_connections",
			Help:      "The number of connections from workspace agents.",
		}, nil),

		ProvisionerJobsActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "coder",
			Name:      "provision_job_active_sum",
			Help:      "The total number of provision jobs in an active state.",
		}, []string{}),

		ProvisionerJobsDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "coder",
			Name:      "provision_job_duration_seconds_sum",
			Help:      "The total run duration of a provision job.",
			Buckets: []float64{
				1,
				5,
				10,
				30,
				60,
				90,
				120,
				180,
				300,
			},
		}, []string{
			"status",
			"job_id",
			"type",
		}),
		Workspaces: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "coder",
			Name:      "workspaces_sum",
			Help:      "The sum of workspaces per user.",
		}, []string{
			"user_id",
			"user_name",
			"template_name",
			"template_id",
			"organization_id",
			"organization_name",
		}),
		WorkspaceResources: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "coder",
			Name:      "workspace_resources_sum",
			Help:      "The created resources in a Coder deployment.",
		}, []string{
			"template_id",
			"template_name",
			"template_version_id",
			"template_version_name",
			"organization_id",
			"organization_name",
			"owner_id",
			"owner_name",
			"workspace_id",
			"workspace_name",
			"type",
		}),

		// Agent connection time
		// Queued jobs

		// When I roll out a new update, I want to see workspaces update.
		//

		// Draw a graph of jobs with the build ID X

		WorkspaceAgents: prometheus.NewGaugeVec(prometheus.GaugeOpts{}, []string{
			"resource_id",
			"resource_name",
			"resource_type",
			"operating_system",
			"architecture",
			"authentication_type",
			"name",
		}),
	}
	return stats
}

type Stats struct {
	AgentConnections,
	AgentSSHSessions,
	AgentReconnectingPTYs *prometheus.GaugeVec

	ProvisionerJobsActive   *prometheus.GaugeVec
	ProvisionerJobsDuration *prometheus.HistogramVec

	ProvisionerDaemonsTotal,
	TemplatesTotal,
	UsersTotal,
	UsersActive prometheus.Gauge
	Workspaces         *prometheus.GaugeVec
	WorkspaceResources *prometheus.GaugeVec
	WorkspaceAgents    *prometheus.GaugeVec
}

func (s Stats) Refresh(ctx context.Context, db database.Store) error {
	var (
		organizations         = map[uuid.UUID]database.Organization{}
		users                 = map[uuid.UUID]database.User{}
		workspaces            = map[uuid.UUID]database.Workspace{}
		workspacesByOwner     = map[uuid.UUID][]uuid.UUID{}
		workspaceBuilds       = map[uuid.UUID]database.WorkspaceBuild{}
		workspaceBuildByJobID = map[uuid.UUID]database.WorkspaceBuild{}
		workspaceBuildJobIDs  = []uuid.UUID{}
		provisionerJobs       = map[uuid.UUID]database.ProvisionerJob{}
		templates             = map[uuid.UUID]database.Template{}
		templateVersions      = map[uuid.UUID]database.TemplateVersion{}

		// Select resources like builds and provisioner jobs that occurred in the past hour.
		createdAfter = database.Now().Add(-time.Hour)
	)

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		dbOrganizations, err := db.GetOrganizations(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get organizations: %w", err)
		}
		for _, org := range dbOrganizations {
			organizations[org.ID] = org
		}
		return nil
	})
	errGroup.Go(func() error {
		dbUsers, err := db.GetUsers(ctx, database.GetUsersParams{})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get users: %w", err)
		}
		for _, user := range dbUsers {
			users[user.ID] = user
		}
		return nil
	})
	errGroup.Go(func() error {
		dbWorkspaces, err := db.GetWorkspaces(ctx, false)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get workspaces: %w", err)
		}
		for _, workspace := range dbWorkspaces {
			workspaces[workspace.ID] = workspace
			ids, valid := workspacesByOwner[workspace.OwnerID]
			if !valid {
				ids = make([]uuid.UUID, 0)
			}
			ids = append(ids, workspace.ID)
			workspacesByOwner[workspace.OwnerID] = ids
		}
		return nil
	})
	errGroup.Go(func() error {
		dbWorkspaceBuilds, err := db.GetWorkspaceBuildsWithoutAfter(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get workspace builds: %w", err)
		}
		for _, workspaceBuild := range dbWorkspaceBuilds {
			workspaceBuilds[workspaceBuild.ID] = workspaceBuild
			workspaceBuildByJobID[workspaceBuild.JobID] = workspaceBuild
			workspaceBuildJobIDs = append(workspaceBuildJobIDs, workspaceBuild.JobID)
		}
		return nil
	})
	errGroup.Go(func() error {
		dbProvisionerJobs, err := db.GetProvisionerJobsAfterCreatedAt(ctx, createdAfter)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get provisioner jobs: %w", err)
		}
		for _, provisionerJob := range dbProvisionerJobs {
			provisionerJobs[provisionerJob.ID] = provisionerJob
		}
		return nil
	})
	errGroup.Go(func() error {
		dbTemplates, err := db.GetTemplates(ctx, false)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get templates: %w", err)
		}
		for _, template := range dbTemplates {
			templates[template.ID] = template
		}
		return nil
	})
	errGroup.Go(func() error {
		dbTemplateVersions, err := db.GetTemplateVersionsAfterCreatedAt(ctx, createdAfter)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get template versions: %w", err)
		}
		for _, templateVersion := range dbTemplateVersions {
			templateVersions[templateVersion.ID] = templateVersion
		}
		return nil
	})
	err := errGroup.Wait()
	if err != nil {
		return err
	}

	// This stores a count of all workspaces with reference to
	// their names, templates, and organization IDs.
	s.Workspaces.Reset()
	for owner, ids := range workspacesByOwner {
		user, valid := users[owner]
		if !valid {
			continue
		}
		for _, workspaceID := range ids {
			workspace, valid := workspaces[workspaceID]
			if !valid {
				continue
			}
			template, valid := templates[workspace.TemplateID]
			if !valid {
				continue
			}
			organization, valid := organizations[workspace.OrganizationID]
			if !valid {
				continue
			}
			s.Workspaces.With(prometheus.Labels{
				"user_id":           user.ID.String(),
				"user_name":         user.Username,
				"template_name":     template.Name,
				"template_id":       template.ID.String(),
				"organization_id":   workspace.OrganizationID.String(),
				"organization_name": organization.Name,
			}).Add(1)
		}
	}

	resources, err := db.GetWorkspaceResourcesByJobIDs(ctx, workspaceBuildJobIDs)
	if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
		return err
	}
	s.WorkspaceResources.Reset()
	for _, resource := range resources {
		build, valid := workspaceBuildByJobID[resource.JobID]
		if !valid {
			continue
		}
		templateVersion, valid := templateVersions[build.TemplateVersionID]
		if !valid {
			continue
		}
		template, valid := templates[templateVersion.TemplateID.UUID]
		if !valid {
			continue
		}
		organization, valid := organizations[template.OrganizationID]
		if !valid {
			continue
		}
		workspace, valid := workspaces[build.WorkspaceID]
		if !valid {
			continue
		}
		user, valid := users[workspace.OwnerID]
		if !valid {
			continue
		}
		s.WorkspaceResources.With(prometheus.Labels{
			"template_version_id":   templateVersion.ID.String(),
			"template_version_name": templateVersion.Name,
			"template_id":           template.ID.String(),
			"template_name":         template.Name,
			"organization_id":       organization.ID.String(),
			"organization_name":     organization.Name,
			"owner_id":              user.ID.String(),
			"owner_name":            user.Username,
			"workspace_id":          workspace.ID.String(),
			"workspace_name":        workspace.Name,
			"type":                  resource.Type,
		}).Add(1)
	}

	return nil
}

type pingUsers struct {
	ID            uuid.UUID
	LastAuth      time.Time
	LastLoginType string
}

type pingAgent struct {
	ID                   uuid.UUID
	ResourceID           uuid.UUID
	AuthType             string
	EnvironmentVariables bool
	Directory            bool
	OperatingSystem      string
	Architecture         string
	StartupScript        bool
}

type pingResources struct {
	ID         uuid.UUID
	TemplateID uuid.UUID
	Type       string
	Transition string
}

type pingTemplates struct {
	ID          uuid.UUID
	Provisioner string
}

type pingWorkspaces struct {
}

type Ping struct {
	SiteIdentifier            string
	ReplicaIdentifier         string
	InstallerEmail            string
	Version                   string
	DeploymentMethod          string
	DeploymentOperatingSystem string
	DevelopmentMode           bool
	Localhost                 bool
	STUN                      bool
	TLS                       bool
	OAuth2Github              bool
}

func durationToFloatMs(d time.Duration) float64 {
	return float64(d.Milliseconds())
}
