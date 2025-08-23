CREATE TABLE IF NOT EXISTS stores_socials (
    store_id INTEGER REFERENCES stores(id) ON DELETE CASCADE,
    social_id INTEGER REFERENCES socials(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    PRIMARY KEY (store_id, social_id)
);