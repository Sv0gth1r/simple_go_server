-- +goose Up
ALTER TABLE Users
ADD COLUMN hashed_password TEXT NOT NULL DEFAULT 'unset';

-- +goose Down
ALTER TABLE Users
DROP COLUMN hashed_password;
