-- Skema awal Financial Tracker.
--
-- Uang disimpan sebagai bigint Rupiah (tanpa desimal) — tidak pernah float.
-- Semua tabel punya owner_id: saat ini satu API key = satu pemilik, nanti
-- bisa diganti login multi-user tanpa mengubah skema.

CREATE TABLE transactions (
    id          uuid        PRIMARY KEY,
    owner_id    text        NOT NULL,
    type        text        NOT NULL CHECK (type IN ('income', 'expense')),
    amount      bigint      NOT NULL CHECK (amount > 0),
    category    text        NOT NULL,
    date        date        NOT NULL,
    note        text,
    created_at  timestamptz NOT NULL
);
-- List terbaru dulu + hitung pemakaian budget per kategori & rentang tanggal.
CREATE INDEX transactions_owner_date_idx ON transactions (owner_id, date DESC, created_at DESC);
CREATE INDEX transactions_owner_category_date_idx ON transactions (owner_id, category, date)
    WHERE type = 'expense';

-- Satu budget per kategori pengeluaran. Pemakaian TIDAK disimpan; dihitung
-- dari transactions supaya tidak pernah selisih.
CREATE TABLE budgets (
    id            uuid        PRIMARY KEY,
    owner_id      text        NOT NULL,
    category      text        NOT NULL,
    period        text        NOT NULL CHECK (period IN ('weekly', 'monthly')),
    limit_amount  bigint      NOT NULL CHECK (limit_amount > 0),
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL,
    UNIQUE (owner_id, category)
);

CREATE TABLE challenges (
    id                uuid        PRIMARY KEY,
    owner_id          text        NOT NULL,
    title             text        NOT NULL,
    temptation_name   text        NOT NULL,
    temptation_price  bigint      NOT NULL CHECK (temptation_price > 0),
    goal_name         text        NOT NULL,
    goal_price        bigint      NOT NULL CHECK (goal_price > 0),
    target_amount     bigint      NOT NULL CHECK (target_amount > 0),
    deadline          date        NOT NULL,
    coin_reward       integer     NOT NULL CHECK (coin_reward > 0),
    saved_amount      bigint      NOT NULL DEFAULT 0 CHECK (saved_amount >= 0),
    status            text        NOT NULL CHECK (status IN ('active', 'success', 'failed')),
    completed_at      timestamptz,
    created_at        timestamptz NOT NULL
);
CREATE INDEX challenges_owner_created_idx ON challenges (owner_id, created_at DESC);

CREATE TABLE challenge_logs (
    id            uuid        PRIMARY KEY,
    challenge_id  uuid        NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    kind          text        NOT NULL CHECK (kind IN ('skip', 'slip')),
    amount        bigint      NOT NULL CHECK (amount >= 0),
    date          date        NOT NULL,
    created_at    timestamptz NOT NULL
);
CREATE INDEX challenge_logs_challenge_idx ON challenge_logs (challenge_id, created_at DESC);

-- Riwayat koin SaveCoin. Saldo = SUM(amount). Koin tetap ada walau challenge
-- dihapus (challenge_id jadi NULL).
CREATE TABLE coin_entries (
    id            uuid        PRIMARY KEY,
    owner_id      text        NOT NULL,
    amount        integer     NOT NULL,
    reason        text        NOT NULL,
    challenge_id  uuid        REFERENCES challenges (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL
);
CREATE INDEX coin_entries_owner_created_idx ON coin_entries (owner_id, created_at DESC);
-- Pengaman: satu challenge hanya bisa membayar koin sekali.
CREATE UNIQUE INDEX coin_entries_challenge_once_idx ON coin_entries (challenge_id)
    WHERE challenge_id IS NOT NULL;
