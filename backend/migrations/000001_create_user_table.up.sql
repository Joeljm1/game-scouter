CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    name text NOT NULL,
    email citext UNIQUE NOT NULL,
    password_hash bytea,
    activated bool NOT NULL,
    version integer NOT NULL DEFAULT 1
);

INSERT INTO users(id,name,email,password_hash,activated) VALUES (0,'Anon','','\xDEADBEEF',true);
