-- +goose Up
CREATE INDEX idx_created_at ON urls (created_at DESC);
-- +goose Down
DROP INDEX idx_created_at;
