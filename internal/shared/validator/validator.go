// Package validator = pengumpul error validasi per field.
//
//	v := validator.New()
//	v.Check(p.Amount > 0, "amount", "must be greater than 0")
//	return v.Err() // nil, atau apperror Invalid (→ 422) berisi semua field
//
// Pesan ditulis dalam bahasa Inggris karena tampil langsung di UI vault.
package validator

import (
	"errors"

	"financial-tracker/internal/shared/apperror"
)

// ErrValidation = pesan umum saat ada field yang tidak valid.
var ErrValidation = errors.New("Some fields are invalid")

// Validator menyimpan pesan pertama per field.
type Validator struct {
	fields map[string]string
}

func New() *Validator { return &Validator{fields: map[string]string{}} }

// Check mencatat msg untuk field jika ok == false.
func (v *Validator) Check(ok bool, field, msg string) {
	if ok {
		return
	}
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = msg
	}
}

// Err mengembalikan apperror Invalid berisi semua field, atau nil.
func (v *Validator) Err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return apperror.Invalid(ErrValidation, v.fields)
}
