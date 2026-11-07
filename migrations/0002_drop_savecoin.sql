-- SaveCoin (SVC) dihapus: challenge tidak lagi memberi koin, dompet tidak ada.
DROP TABLE IF EXISTS coin_entries;
ALTER TABLE challenges DROP COLUMN IF EXISTS coin_reward;
