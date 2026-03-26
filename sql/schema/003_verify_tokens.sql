-- +goose up
CREATE TABLE verify_tokens(
    token TEXT PRIMARY KEY NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN);

-- +goose down
DROP TABLE verify_tokens;