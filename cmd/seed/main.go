// Command seed mengisi database dengan data contoh untuk dicoba di josh-vault.
//
//	go run ./cmd/seed          # isi data contoh (aman dijalankan ulang, tidak dobel)
//	go run ./cmd/seed -clear   # hapus HANYA data contoh
//
// Tanggal dibuat relatif terhadap hari ini (zona TIMEZONE), jadi kapan pun
// dijalankan, minggu ini tetap punya budget yang "hampir habis" dan "lewat".
// ID dibuat deterministik (UUID v5 dari nama), sehingga -clear tidak pernah
// menyentuh data yang kamu masukkan sendiri.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	_ "time/tzdata"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"financial-tracker/internal/shared/clock"
	"financial-tracker/migrations"
	"financial-tracker/platform/config"
	"financial-tracker/platform/postgres"
)

// seedNS = namespace UUID v5 untuk semua data contoh.
var seedNS = uuid.MustParse("6f1c2a7e-5b8d-4c3e-9a10-5eed0da7a001")

func id(name string) string { return uuid.NewSHA1(seedNS, []byte(name)).String() }

func main() {
	clear := flag.Bool("clear", false, "hapus data contoh saja")
	flag.Parse()
	if err := run(*clear); err != nil {
		fmt.Fprintln(os.Stderr, "seed gagal:", err)
		os.Exit(1)
	}
}

type txRow struct {
	key, typ, category, note string
	amount                   int64
	date                     time.Time
}

type logRow struct {
	kind string
	date time.Time
}

type challengeRow struct {
	key, title, tName, gName, status string
	tPrice, gPrice, target           int64
	deadline                         time.Time
	logs                             []logRow
	createdDaysAgo                   int
}

type budgetRow struct {
	category, period string
	limit            int64
}

func run(clearOnly bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := postgres.Connect(ctx, cfg.MigrationURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := postgres.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}

	clk := clock.New(cfg.Location)
	today := clk.Today()
	now := clk.Now()
	day := func(offset int) time.Time { return today.AddDate(0, 0, offset) }
	// Senin minggu ini.
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	week := func(i int) time.Time { return monday.AddDate(0, 0, i) }
	monthStart := clock.Date(today.Year(), today.Month(), 1)
	month := func(d int) time.Time { return monthStart.AddDate(0, 0, d-1) }
	prevMonth := func(d int) time.Time { return monthStart.AddDate(0, -1, d-1) }

	// ---- Transaksi ----
	txs := []txRow{
		// Pemasukan
		{"salary-prev", "income", "Salary", "Monthly salary", 8_500_000, prevMonth(1)},
		{"salary", "income", "Salary", "Monthly salary", 8_500_000, month(1)},
		{"freelance", "income", "Freelance", "Landing page project", 1_500_000, day(-2)},
		{"gift", "income", "Gift", "Birthday money", 250_000, prevMonth(18)},

		// Food & Drinks minggu ini → ~86% dari Rp 350.000 (hampir habis)
		{"food-w0", "expense", "Food & Drinks", "Nasi padang", 45_000, week(0)},
		{"food-w1", "expense", "Food & Drinks", "Yogurt at the mall", 70_000, week(1)},
		{"food-w2", "expense", "Food & Drinks", "Bakso", 38_000, week(2)},
		{"food-w3", "expense", "Food & Drinks", "Dinner with friends", 95_000, week(3)},
		{"food-w4", "expense", "Food & Drinks", "Ayam geprek", 52_000, week(4)},
		// Snacks minggu ini → lewat dari Rp 100.000
		{"snack-w0", "expense", "Snacks", "Chips & soda", 25_000, week(0)},
		{"snack-w2", "expense", "Snacks", "Bubble tea", 45_000, week(2)},
		{"snack-w3", "expense", "Snacks", "Martabak manis", 40_000, week(3)},

		// Minggu lalu
		{"food-pw1", "expense", "Food & Drinks", "Groceries", 180_000, monday.AddDate(0, 0, -6)},
		{"food-pw2", "expense", "Food & Drinks", "Sate ayam", 40_000, monday.AddDate(0, 0, -4)},
		{"snack-pw", "expense", "Snacks", "Iced latte", 45_000, monday.AddDate(0, 0, -3)},

		// Bulan ini
		{"transport-1", "expense", "Transport", "Fuel", 100_000, month(2)},
		{"transport-2", "expense", "Transport", "Ojek online", 35_000, day(-1)},
		{"bills-1", "expense", "Bills", "Electricity", 350_000, month(3)},
		{"bills-2", "expense", "Bills", "Internet", 330_000, month(3)},
		{"shop-1", "expense", "Shopping", "New t-shirt", 150_000, day(-3)},
		{"ent-1", "expense", "Entertainment", "Netflix", 65_000, month(1)},
		{"ent-2", "expense", "Entertainment", "Cinema", 100_000, day(-4)},
		{"health-1", "expense", "Health", "Vitamins", 85_000, day(-5)},

		// Bulan lalu
		{"bills-prev", "expense", "Bills", "Electricity", 320_000, prevMonth(3)},
		{"rent-prev", "expense", "Bills", "Boarding house", 1_500_000, prevMonth(5)},
		{"shop-prev", "expense", "Shopping", "Running shoes", 650_000, prevMonth(12)},
		{"food-prev", "expense", "Food & Drinks", "Groceries", 420_000, prevMonth(15)},
		{"transport-prev", "expense", "Transport", "Fuel", 200_000, prevMonth(20)},
	}

	budgets := []budgetRow{
		{"Food & Drinks", "weekly", 350_000}, // hampir habis
		{"Snacks", "weekly", 100_000},        // lewat
		{"Transport", "monthly", 500_000},    // aman
		{"Entertainment", "monthly", 300_000},
	}

	challenges := []challengeRow{
		{
			key: "yogurt", title: "Skip the yogurt", tName: "Yogurt", tPrice: 70_000,
			gName: "Dinner", gPrice: 35_000, target: 350_000, status: "active",
			deadline: day(14), createdDaysAgo: 6,
			logs: []logRow{{"skip", day(-5)}, {"skip", day(-3)}, {"slip", day(-2)}, {"skip", day(-1)}},
		},
		{
			key: "boba", title: "No bubble tea this month", tName: "Bubble tea", tPrice: 30_000,
			gName: "Breakfast", gPrice: 15_000, target: 150_000, status: "active",
			deadline: day(24), createdDaysAgo: 3,
			logs: []logRow{{"skip", day(-2)}},
		},
		{
			key: "coffee", title: "No fancy coffee", tName: "Iced latte", tPrice: 45_000,
			gName: "Lunch", gPrice: 25_000, target: 135_000, status: "success",
			deadline: day(5), createdDaysAgo: 12,
			logs: []logRow{{"skip", day(-10)}, {"slip", day(-8)}, {"skip", day(-6)}, {"skip", day(-3)}},
		},
		{
			key: "delivery", title: "Cook instead of delivery", tName: "Food delivery", tPrice: 85_000,
			gName: "Groceries for 2 days", gPrice: 100_000, target: 425_000, status: "failed",
			deadline: day(-4), createdDaysAgo: 25,
			logs: []logRow{{"skip", day(-20)}, {"slip", day(-15)}, {"skip", day(-9)}},
		},
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Selalu hapus dulu data contoh lama → seed = "reset ke data contoh terbaru".
	if err := clearSeed(ctx, tx, cfg.OwnerID, txs, budgets, challenges); err != nil {
		return err
	}
	if clearOnly {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		fmt.Println("data contoh dihapus")
		return nil
	}

	inserted := 0
	for _, t := range txs {
		if t.date.After(today) {
			continue // awal minggu: jangan buat transaksi di masa depan
		}
		if _, err := tx.Exec(ctx, `INSERT INTO transactions (id, owner_id, type, amount, category, date, note, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			id("tx:"+t.key), cfg.OwnerID, t.typ, t.amount, t.category, t.date, t.note, t.date.Add(12*time.Hour)); err != nil {
			return fmt.Errorf("transaksi %s: %w", t.key, err)
		}
		inserted++
	}

	for _, b := range budgets {
		// Kategori yang sudah punya budget buatanmu sendiri dilewati.
		if _, err := tx.Exec(ctx, `INSERT INTO budgets (id, owner_id, category, period, limit_amount, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6) ON CONFLICT (owner_id, category) DO NOTHING`,
			id("budget:"+b.category), cfg.OwnerID, b.category, b.period, b.limit, now); err != nil {
			return fmt.Errorf("budget %s: %w", b.category, err)
		}
	}

	for _, c := range challenges {
		cid := id("challenge:" + c.key)
		var saved int64
		for _, l := range c.logs {
			if l.kind == "skip" {
				saved += c.tPrice
			}
		}
		var completedAt *time.Time
		if c.status == "success" {
			t := c.logs[len(c.logs)-1].date.Add(13 * time.Hour)
			completedAt = &t
		}
		if _, err := tx.Exec(ctx, `INSERT INTO challenges (id, owner_id, title, temptation_name, temptation_price,
			goal_name, goal_price, target_amount, deadline, saved_amount, status, completed_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			cid, cfg.OwnerID, c.title, c.tName, c.tPrice, c.gName, c.gPrice, c.target, c.deadline,
			saved, c.status, completedAt, now.AddDate(0, 0, -c.createdDaysAgo)); err != nil {
			return fmt.Errorf("challenge %s: %w", c.key, err)
		}
		for i, l := range c.logs {
			amount := int64(0)
			if l.kind == "skip" {
				amount = c.tPrice
			}
			if _, err := tx.Exec(ctx, `INSERT INTO challenge_logs (id, challenge_id, kind, amount, date, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				id(fmt.Sprintf("log:%s:%d", c.key, i)), cid, l.kind, amount, l.date, l.date.Add(13*time.Hour)); err != nil {
				return fmt.Errorf("log %s: %w", c.key, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("data contoh dibuat: %d transaksi, %d budget, %d challenge\n",
		inserted, len(budgets), len(challenges))
	return nil
}

func clearSeed(ctx context.Context, tx pgx.Tx, owner string,
	txs []txRow, budgets []budgetRow, challenges []challengeRow,
) error {
	var txIDs, budgetIDs, challengeIDs []string
	for _, t := range txs {
		txIDs = append(txIDs, id("tx:"+t.key))
	}
	for _, b := range budgets {
		budgetIDs = append(budgetIDs, id("budget:"+b.category))
	}
	for _, c := range challenges {
		challengeIDs = append(challengeIDs, id("challenge:"+c.key))
	}
	stmts := []struct {
		sql string
		ids []string
	}{
		{`DELETE FROM challenges WHERE owner_id = $1 AND id = ANY($2)`, challengeIDs}, // logs ikut (CASCADE)
		{`DELETE FROM budgets WHERE owner_id = $1 AND id = ANY($2)`, budgetIDs},
		{`DELETE FROM transactions WHERE owner_id = $1 AND id = ANY($2)`, txIDs},
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s.sql, owner, s.ids); err != nil {
			return err
		}
	}
	return nil
}
