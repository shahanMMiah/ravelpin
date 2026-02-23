-- +goose up
ALTER TABLE users
ADD role TEXT NOT NULL;

-- +goose down
ALTER TABLE users
DROP COLUMN role;

