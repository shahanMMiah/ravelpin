-- +goose up
CREATE TABLE ravel_hashes (
    id SERIAL PRIMARY KEY,
    image_path TEXT NOT NULL,
    ravel_post TEXT NOT NULL,
    -- Store the full 64-bit hash for final distance calculation
    full_hash BIGINT NOT NULL,
    -- The four 16-bit segments for indexing
    hash_part_1 SMALLINT NOT NULL,
    hash_part_2 SMALLINT NOT NULL,
    hash_part_3 SMALLINT NOT NULL,
    hash_part_4 SMALLINT NOT NULL
);

CREATE INDEX idx_hash_part_1 ON ravel_hashes (hash_part_1);
CREATE INDEX idx_hash_part_2 ON ravel_hashes (hash_part_2);
CREATE INDEX idx_hash_part_3 ON ravel_hashes (hash_part_3);
CREATE INDEX idx_hash_part_4 ON ravel_hashes (hash_part_4);


-- +goose down
DROP TABLE ravel_hashes;