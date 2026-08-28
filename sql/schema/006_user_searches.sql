-- +goose up
 CREATE TABLE user_search(
     id UUID PRIMARY KEY NOT NULL,
     created_at TIMESTAMP NOT NULL,
     updated_at TIMESTAMP NOT NULL,
     search_img TEXT NOT NULL);

 CREATE TABLE search_result(
     id UUID PRIMARY KEY NOT NULL,
     name TEXT NOT NULL,
     permalink TEXT NOT NULL,
     image_path TEXT NOT NULL,
     search_id UUID REFERENCES user_search(id) ON DELETE CASCADE NOT NULL);

-- +goose down
DROP TABLE search_result;
DROP TABLE user_search;
