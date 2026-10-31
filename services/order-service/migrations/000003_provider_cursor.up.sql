CREATE TABLE provider_cursors(provider text PRIMARY KEY,position bigint NOT NULL DEFAULT 0 CHECK(position>=0));
