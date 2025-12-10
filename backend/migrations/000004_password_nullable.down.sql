UPDATE users
SET password_hash = '\x00'
WHERE password_hash IS NULL;

ALTER TABLE users
ALTER COLUMN password_hash  SET NOT NULL;

