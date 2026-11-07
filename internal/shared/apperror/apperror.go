// Package apperror memberi "jenis" pada error supaya layer HTTP bisa memilih
// status code tanpa mengenal error milik tiap domain.
//
//	domain  → error biasa (sentinel, mis. ErrNotFound)
//	usecase → membungkus: apperror.NotFound(domain.ErrNotFound)
//	handler → httpserver.WriteError membaca Kind → 404
//
// Unwrap tetap mengembalikan error asli, jadi errors.Is masih bekerja.
package apperror

import "errors"

// Kind = kategori error yang relevan untuk client.
type Kind int

const (
	KindInternal Kind = iota // bug / infrastruktur → 500
	KindInvalid              // input tidak valid → 422
	KindNotFound             // resource tidak ada → 404
	KindConflict             // state tidak mengizinkan aksi → 409
)

// Error membungkus error asli + jenisnya. Fields = detail validasi per field.
type Error struct {
	Kind   Kind
	Err    error
	Fields map[string]string
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func NotFound(err error) error { return &Error{Kind: KindNotFound, Err: err} }
func Conflict(err error) error { return &Error{Kind: KindConflict, Err: err} }

// Invalid menandai err sebagai input tidak valid, opsional dengan detail field.
func Invalid(err error, fields map[string]string) error {
	return &Error{Kind: KindInvalid, Err: err, Fields: fields}
}

// KindOf mengembalikan jenis error; error tanpa jenis dianggap KindInternal.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// FieldsOf mengembalikan detail field (bisa nil).
func FieldsOf(err error) map[string]string {
	var e *Error
	if errors.As(err, &e) {
		return e.Fields
	}
	return nil
}
