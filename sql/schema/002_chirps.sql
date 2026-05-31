-- +goose Up
CREATE TABLE Chirps(
	id UUID PRIMARY KEY,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	body TEXT NOT NULL,
	user_id UUID,
	CONSTRAINT fk_user_id
		FOREIGN KEY (user_id)
		REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps
