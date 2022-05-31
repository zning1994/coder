-- Setting a default max_ttl of seven days.
ALTER TABLE ONLY templates ADD COLUMN max_ttl BIGINT NOT NULL DEFAULT 604800000000000;
