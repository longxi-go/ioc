CREATE TABLE IF NOT EXISTS orders (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 customer TEXT NOT NULL,
 coffee TEXT NOT NULL,
 quantity INTEGER NOT NULL CHECK(quantity BETWEEN 1 AND 100),
 unit_price INTEGER NOT NULL CHECK(unit_price > 0),
 total INTEGER NOT NULL CHECK(total = quantity * unit_price),
 status TEXT NOT NULL CHECK(status IN ('placed','preparing','ready','collected','cancelled')),
 version INTEGER NOT NULL CHECK(version > 0),
 created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL
);
