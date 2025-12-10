ALTER TABLE users 
ALTER COLUMN password_hash  DROP NOT NULL;

-- this is to revert later see down migration  to understand
UPDATE users
SET password_hash = NULL
WHERE password_hash IS NOT NULL
  AND length(password_hash  ) = 0;

