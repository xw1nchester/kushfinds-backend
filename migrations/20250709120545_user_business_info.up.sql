 
CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS states (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    country_id INTEGER,
    FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    state_id INTEGER,
    FOREIGN KEY (state_id) REFERENCES states(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS business_industries (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

ALTER TABLE users
    ADD COLUMN is_admin BOOLEAN DEFAULT false NOT NULL,
    ADD COLUMN country_id INTEGER,
    ADD COLUMN state_id INTEGER,
    ADD COLUMN region_id INTEGER,
    ADD COLUMN age INTEGER,
    ADD COLUMN phone_number TEXT;

ALTER TABLE users ADD CONSTRAINT fk_country_id FOREIGN KEY (country_id) REFERENCES countries(id);
ALTER TABLE users ADD CONSTRAINT fk_state_id FOREIGN KEY (state_id) REFERENCES states(id);
ALTER TABLE users ADD CONSTRAINT fk_region_id FOREIGN KEY (region_id) REFERENCES regions(id);

CREATE TABLE IF NOT EXISTS business_profiles (
    user_id INTEGER PRIMARY KEY,
    business_industry_id INTEGER,
    business_name TEXT,
    country_id INTEGER,
    state_id INTEGER,
    region_id INTEGER,
    email TEXT,
    phone_number TEXT,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (business_industry_id) REFERENCES business_industries(id) ON DELETE SET NULL,
    FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE SET NULL,
    FOREIGN KEY (state_id) REFERENCES states(id) ON DELETE SET NULL,
    FOREIGN KEY (region_id) REFERENCES regions(id) ON DELETE SET NULL
);