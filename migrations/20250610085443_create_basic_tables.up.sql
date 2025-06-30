CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    username TEXT NULL,
    first_name TEXT NULL,
    last_name TEXT NULL,
    avatar TEXT NULL,
    password_hash TEXT NULL,
    is_verified BOOLEAN DEFAULT false NOT NULL
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'code_type') THEN
        CREATE TYPE code_type AS ENUM ('verification', 'recovery_password', 'change_password');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS codes (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    type code_type,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    token TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);