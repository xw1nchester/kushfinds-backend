CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    username VARCHAR(255) NULL,
    first_name VARCHAR(255) NULL,
    last_name VARCHAR(255) NULL,
    avatar VARCHAR(255) NULL,
    password_hash VARCHAR(255) NULL,
    is_verified BOOLEAN DEFAULT false NOT NULL
);

CREATE TYPE code_type AS ENUM ('verification', 'recovery_password', 'change_password');

CREATE TABLE codes (
    id SERIAL PRIMARY KEY,
    code VARCHAR(255) NOT NULL,
    type code_type,
    user_id INTEGER REFERENCES users(id) NOT NULL
);

CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL,
    user_agent VARCHAR(255) NOT NULL,
    user_id INTEGER REFERENCES users(id) NOT NULL
);