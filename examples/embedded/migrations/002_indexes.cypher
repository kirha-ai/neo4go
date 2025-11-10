-- +neo4go Up
CREATE INDEX user_email_idx IF NOT EXISTS
FOR (u:User) ON (u.email);

CREATE INDEX user_created_at_idx IF NOT EXISTS
FOR (u:User) ON (u.created_at);

-- +neo4go Down
DROP INDEX user_email_idx IF EXISTS;
DROP INDEX user_created_at_idx IF EXISTS;
