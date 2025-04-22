-- +goose Up
CREATE TABLE chirps (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    body text NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;

