CREATE TABLE IF NOT EXISTS market_sections (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS brands (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    country_id INTEGER,
    market_section_id INTEGER,
    name TEXT,
    email TEXT,
    phone_number TEXT,
    logo TEXT,
    banner TEXT,
    created_at timestamp(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE CASCADE,
    FOREIGN KEY (market_section_id) REFERENCES market_sections(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS brands_market_sub_sections (
    brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
    market_section_id INTEGER REFERENCES market_sections(id) ON DELETE CASCADE,
    PRIMARY KEY (brand_id, market_section_id)
);

CREATE TABLE IF NOT EXISTS brands_states (
    brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
    state_id INTEGER REFERENCES states(id) ON DELETE CASCADE,
    PRIMARY KEY (brand_id, state_id)
);
