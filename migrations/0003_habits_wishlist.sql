-- "Habits I don't have" + wishlist.
--
-- Habit hanya menyimpan biaya & tanggal mulai. Uang yang "tidak jadi keluar"
-- TIDAK disimpan; dihitung client dari cost × hari sejak `since`, jadi selalu
-- up to date tanpa job harian.

CREATE TABLE habits (
    id          uuid        PRIMARY KEY,
    owner_id    text        NOT NULL,
    name        text        NOT NULL,
    icon        text        NOT NULL,
    cost        bigint      NOT NULL CHECK (cost > 0),
    frequency   text        NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    since       date        NOT NULL,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL
);
CREATE INDEX habits_owner_created_idx ON habits (owner_id, created_at);

-- Barang yang ingin dibeli — satuan konversi untuk uang yang tersimpan.
CREATE TABLE wishlist_items (
    id          uuid        PRIMARY KEY,
    owner_id    text        NOT NULL,
    name        text        NOT NULL,
    price       bigint      NOT NULL CHECK (price > 0),
    created_at  timestamptz NOT NULL
);
CREATE INDEX wishlist_items_owner_price_idx ON wishlist_items (owner_id, price);
