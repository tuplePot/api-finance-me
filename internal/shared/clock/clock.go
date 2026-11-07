// Package clock = sumber waktu aplikasi.
//
// "Hari ini" dihitung di zona waktu pengguna (TIMEZONE, default Asia/Jakarta),
// bukan UTC — kalau tidak, transaksi jam 01.00 WIB akan masuk ke hari kemarin.
// Tanggal (tanpa jam) direpresentasikan sebagai time.Time jam 00:00 UTC, sama
// seperti cara pgx membaca kolom `date`.
package clock

import "time"

// Clock memberi waktu sekarang dan tanggal hari ini.
type Clock struct {
	loc *time.Location
	now func() time.Time
}

// New membuat Clock untuk zona waktu loc.
func New(loc *time.Location) Clock {
	return Clock{loc: loc, now: time.Now}
}

// Fixed = Clock dengan waktu tetap, untuk test.
func Fixed(t time.Time, loc *time.Location) Clock {
	return Clock{loc: loc, now: func() time.Time { return t }}
}

// Now = waktu sekarang (UTC), untuk created_at / completed_at.
func (c Clock) Now() time.Time { return c.now().UTC() }

// Today = tanggal hari ini di zona waktu pengguna.
func (c Clock) Today() time.Time {
	y, m, d := c.now().In(c.loc).Date()
	return Date(y, m, d)
}

// Date membuat nilai tanggal (00:00 UTC).
func Date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DateLayout = format tanggal di JSON (yyyy-mm-dd), sama dengan <input type="date">.
const DateLayout = "2006-01-02"

// ParseDate membaca "yyyy-mm-dd".
func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateLayout, s)
}
