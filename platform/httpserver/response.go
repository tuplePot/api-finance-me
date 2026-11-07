package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"financial-tracker/internal/shared/apperror"
)

// maxBodyBytes membatasi ukuran body request (1 MiB).
const maxBodyBytes = 1 << 20

// Envelope = bentuk semua response, sama dengan API lain di josh-vault:
//
//	sukses: { "success": true,  "message": "...", "data": ... }
//	gagal:  { "success": false, "message": "...", "code": "not_found", "fields": {...} }
//
// message tampil langsung di UI, jadi ditulis dalam bahasa Inggris.
type Envelope struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    any               `json:"data"`
	Code    string            `json:"code,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteJSON mengirim v sebagai JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("gagal menulis response JSON", "err", err)
	}
}

// OK mengirim response sukses.
func OK(w http.ResponseWriter, status int, message string, data any) {
	WriteJSON(w, status, Envelope{Success: true, Message: message, Data: data})
}

// Fail mengirim response gagal dengan code tertentu.
func Fail(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, Envelope{Success: false, Message: message, Code: code})
}

// WriteError memilih status dari jenis error. Error tanpa jenis = 500 dan
// pesan aslinya tidak dibocorkan ke client.
func WriteError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	msg := "Something went wrong on the server."

	switch apperror.KindOf(err) {
	case apperror.KindInvalid:
		status, code, msg = http.StatusUnprocessableEntity, "validation_error", err.Error()
	case apperror.KindNotFound:
		status, code, msg = http.StatusNotFound, "not_found", err.Error()
	case apperror.KindConflict:
		status, code, msg = http.StatusConflict, "conflict", err.Error()
	default:
		slog.Error("internal error", "err", err)
	}
	WriteJSON(w, status, Envelope{Success: false, Message: msg, Code: code, Fields: apperror.FieldsOf(err)})
}

// DecodeJSON membaca body JSON ke v. Field tak dikenal ditolak supaya salah
// ketik nama field langsung ketahuan.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return apperror.Invalid(errors.New("Request body is required"), nil)
		}
		return apperror.Invalid(errors.New("Request body is not valid JSON: "+err.Error()), nil)
	}
	return nil
}
