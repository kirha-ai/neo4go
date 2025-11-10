-- +neo4go Up
CREATE CONSTRAINT user_id_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.id IS UNIQUE;

CREATE CONSTRAINT user_email_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.email IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT user_id_unique IF EXISTS;
DROP CONSTRAINT user_email_unique IF EXISTS;
