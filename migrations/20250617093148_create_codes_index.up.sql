CREATE UNIQUE INDEX IF NOT EXISTS idx_codes_type_user_id
ON codes (type, user_id);