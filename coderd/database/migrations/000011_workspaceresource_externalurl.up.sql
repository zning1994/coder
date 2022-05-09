ALTER TABLE ONLY workspace_resources
	ADD COLUMN IF NOT EXISTS external_url varchar(1024);
