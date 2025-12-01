-- +goose Up
CREATE TABLE urls (
    id BIGSERIAL PRIMARY KEY,
    long_url TEXT NOT NULL,
    short_url VARCHAR(10) NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(), 
    click_count INTEGER NOT NULL DEFAULT 0,
    last_accessed_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NULL,

    CONSTRAINT unique_long_url UNIQUE (long_url), 
    CONSTRAINT unique_short_url UNIQUE (short_url)
);

CREATE INDEX idx_short_url ON urls (short_url);

-- +goose Down
DROP TABLE urls;
