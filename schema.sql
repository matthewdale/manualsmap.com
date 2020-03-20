CREATE TABLE map_blocks (
    id SERIAL PRIMARY KEY,
    latitude NUMERIC NOT NULL,
    longitude NUMERIC NOT NULL,
    UNIQUE (longitude, latitude)
);

CREATE TABLE cars (
    license_hash TEXT PRIMARY KEY,
    map_block_id INTEGER NOT NULL,
    year INTEGER NOT NULL,
    brand TEXT NOT NULL,
    model TEXT NOT NULL,
    trim TEXT NOT NULL,
    color TEXT NOT NULL,
    image_url TEXT NOT NULL
);
