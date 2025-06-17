CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    username VARCHAR(255) NULL,
    first_name VARCHAR(255) NULL,
    last_name VARCHAR(255) NULL,
    avatar VARCHAR(255) NULL,
    password_hash VARCHAR(255) NULL,
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
    code VARCHAR(255) NOT NULL,
    type code_type,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL,
    user_agent VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);