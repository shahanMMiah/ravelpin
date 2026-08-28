-- +goose up
ALTER TABLE user_search
ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL;

-- +goose down
DROP COLUMN user_id;
