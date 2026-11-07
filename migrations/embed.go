// Package migrations menyimpan file SQL skema database.
//
// `//go:embed` memasukkan file .sql ke dalam binary saat build, jadi folder
// migrations tidak perlu ikut dibawa saat deploy.
package migrations

import "embed"

// FS berisi semua file *.sql, diurutkan berdasarkan nama (0001_, 0002_, ...).
//
//go:embed *.sql
var FS embed.FS
