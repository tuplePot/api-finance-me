// Package owner membawa "pemilik data" lewat context request.
//
// Saat ini API memakai satu API key = satu pemilik (OWNER_ID). Semua tabel
// tetap punya kolom owner_id, jadi nanti cukup ganti middleware auth (mis.
// JWT) tanpa mengubah usecase / repository.
package owner

import "context"

type ctxKey struct{}

// With menyimpan id pemilik ke context.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From mengambil id pemilik. String kosong = tidak ada (seharusnya tidak
// terjadi di route yang dilindungi middleware auth).
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
