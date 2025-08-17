ALTER TABLE
    brands
ADD
    COLUMN is_published BOOLEAN DEFAULT false NOT NULL;

CREATE TABLE IF NOT EXISTS brands_documents (
    brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    PRIMARY KEY (brand_id, url)
);

CREATE TABLE IF NOT EXISTS socials (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    icon TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS brands_socials (
    brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
    social_id INTEGER REFERENCES socials(id) ON DELETE CASCADE,
    PRIMARY KEY (brand_id, social_id)
);