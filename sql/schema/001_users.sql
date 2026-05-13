-- +goose Up

create table users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    hashed_password text not null,
    created_at timestamp not null default NOW(),
    modified_at timestamp not null default NOW()
);

-- +goose Down

DROP TABLE users;