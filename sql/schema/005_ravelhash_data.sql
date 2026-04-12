-- +goose up
ALTER TABLE ravel_hashes
ADD COLUMN ravel_id INTEGER NOT NULL, 
ADD COLUMN permalink TEXT NOT NULL, 
ADD COLUMN post_name TEXT NOT NULL;

-- +goose down
ALTER TABLE ravel_hashes
DROP COLUMN ravel_id, 
DROP COLUMN permalink, 
DROP COLUMN post_name;