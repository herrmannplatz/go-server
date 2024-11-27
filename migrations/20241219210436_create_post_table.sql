-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE post;