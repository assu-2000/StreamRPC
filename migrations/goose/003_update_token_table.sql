-- +goose Up
ALTER TABLE refresh_tokens ADD COLUMN revoked BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION hash_token(token TEXT) RETURNS TEXT AS $$
BEGIN
RETURN encode(sha256(token::bytea), 'hex');
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd