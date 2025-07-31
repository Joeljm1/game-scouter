CREATE TABLE token (
	user_id bigint REFERENCES users ON DELETE CASCADE,
	hash bytea PRIMARY KEY,
	expiry timestamp(0) with time zone NOT NULL,
	scope text NOT NULL,
	data bytea,
)
