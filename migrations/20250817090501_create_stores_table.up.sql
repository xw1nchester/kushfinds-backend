CREATE TABLE IF NOT EXISTS store_types (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS stores (
    id SERIAL PRIMARY KEY,
    brand_id INTEGER,
    name TEXT,
    banner TEXT,
    description TEXT,
    country_id INTEGER,
    state_id INTEGER,
    region_id INTEGER,
    street TEXT,
    house TEXT,
    post_code TEXT,
    email TEXT,
    phone_number TEXT,
    store_type_id INTEGER,
    delivery_price INTEGER,
    minimal_order_price INTEGER,
    delivery_distance INTEGER,
    is_published BOOLEAN DEFAULT false NOT NULL,
    FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE CASCADE,
    FOREIGN KEY (state_id) REFERENCES states(id) ON DELETE CASCADE,
    FOREIGN KEY (region_id) REFERENCES regions(id) ON DELETE CASCADE,
    FOREIGN KEY (store_type_id) REFERENCES store_types(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stores_pictures (
    store_id INTEGER REFERENCES stores(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    PRIMARY KEY (store_id, url)
);