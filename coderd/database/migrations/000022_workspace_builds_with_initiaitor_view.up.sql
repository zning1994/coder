-- This view adds the initiator name to the query for UI purposes.
-- Showing the initiator user ID is not very friendly.
CREATE VIEW workspace_build_with_names AS
SELECT
    -- coalesce is used because technically the joins do not guarantee a value.
    -- If we setup proper foreign keys, we can remove the coalesce.
	coalesce(initiator_user.username, 'unknown') AS initiator_username,
	coalesce(workspaces.owner_id, '00000000-00000000-00000000-00000000') AS owner_id,
	coalesce(owner_user.username, 'unknown') AS owner_name,
	coalesce(workspaces.name, 'unknown') AS workspace_name,
	coalesce(templates.id, '00000000-00000000-00000000-00000000') AS template_id,
	coalesce(templates.name, 'unknown') AS template_name,
	coalesce(templates.active_version_id, '00000000-00000000-00000000-00000000') AS template_active_version,
	workspace_builds.*
FROM workspace_builds
		 LEFT JOIN users AS initiator_user ON workspace_builds.initiator_id = initiator_user.id
		 LEFT JOIN workspaces ON workspaces.id = workspace_builds.workspace_id
		 LEFT JOIN users AS owner_user ON workspaces.owner_id = owner_user.id
		 LEFT JOIN templates ON workspaces.template_id = templates.id;
