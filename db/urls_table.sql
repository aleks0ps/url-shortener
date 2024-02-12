CREATE TABLE IF NOT EXISTS urls (
  uuid SERIAL PRIMARY KEY,
  short_url TEXT NOT NULL,
  original_url TEXT NOT NULL
);
