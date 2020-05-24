CREATE TABLE map_blocks (
    id SERIAL PRIMARY KEY,
    latitude NUMERIC NOT NULL,
    longitude NUMERIC NOT NULL,
    UNIQUE (longitude, latitude)
);

CREATE TYPE status_t AS ENUM('pending', 'approved', 'rejected');
CREATE TABLE images (
    public_id TEXT PRIMARY KEY,
    format TEXT NOT NULL,
    status status_t NOT NULL DEFAULT 'pending',
    created timestamp NOT NULL DEFAULT NOW(),
    updated timestamp NOT NULL DEFAULT NOW()
);

CREATE TABLE cars (
    id SERIAL PRIMARY KEY,
    map_block_id INTEGER NOT NULL,
    year INTEGER NOT NULL,
    make TEXT NOT NULL,
    model TEXT NOT NULL,
    trim TEXT NOT NULL,
    color TEXT NOT NULL,
    images_public_id TEXT,
    created timestamp NOT NULL DEFAULT NOW()
);
