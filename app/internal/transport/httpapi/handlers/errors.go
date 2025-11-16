package handlers

import (
	"avito/train-assignment/app/internal/domain"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// apiError описывает машину состояний маппинга ошибок домена в контракт API.
type apiError struct {
	HTTP   int
	Code   string
	Detail string
}

// writeError сериализует ответ в формат ErrorResponse из openapi.yml.
func writeError(w http.ResponseWriter, ae apiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ae.HTTP)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    ae.Code,
			"message": ae.Detail,
		},
	})
}

// mapError принимает исходную ошибку и возвращает готовый apiError.
func mapError(err error) apiError {
	if err == nil {
		return apiError{HTTP: http.StatusOK, Code: "OK", Detail: "ok"}
	}

	// Доменные ошибки.
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return apiError{HTTP: http.StatusNotFound, Code: "NOT_FOUND", Detail: "resource not found"}
	case errors.Is(err, domain.ErrPRMerged):
		// При повторном merge требование — идемпотентность: не ошибка для повторов.
		return apiError{HTTP: http.StatusOK, Code: "OK", Detail: "already merged"}
	case errors.Is(err, domain.ErrNoCandidate):
		return apiError{HTTP: http.StatusConflict, Code: "NO_CANDIDATE", Detail: "no active candidate available"}
	case errors.Is(err, domain.ErrNotAssigned):
		return apiError{HTTP: http.StatusConflict, Code: "NOT_ASSIGNED", Detail: "reviewer not assigned"}
	}

	// SQL: no rows -> NOT_FOUND.
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{HTTP: http.StatusNotFound, Code: "NOT_FOUND", Detail: "resource not found"}
	}

	// Конфликт уникальности -> TEAM_EXISTS / PR_EXISTS в зависимости от контекста.
	// Здесь нейтрально возвращаем 409 CONFLICT; конкретный код можно уточнить на уровне хендлера.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" { // unique_violation
			return apiError{HTTP: http.StatusConflict, Code: "CONFLICT", Detail: "unique constraint violation"}
		}
	}

	// По умолчанию — INTERNAL.
	return apiError{HTTP: http.StatusInternalServerError, Code: "INTERNAL", Detail: "internal error"}
}

// Helpers для хендлеров:

// NotFound шорткат.
func NotFound(w http.ResponseWriter) {
	writeError(w, apiError{HTTP: http.StatusNotFound, Code: "NOT_FOUND", Detail: "resource not found"})
}

// Conflict код с конкретным ErrorResponse code.
func Conflict(w http.ResponseWriter, code, msg string) {
	writeError(w, apiError{HTTP: http.StatusConflict, Code: code, Detail: msg})
}

// BadRequest 400.
func BadRequest(w http.ResponseWriter, msg string) {
	writeError(w, apiError{HTTP: http.StatusBadRequest, Code: "BAD_REQUEST", Detail: msg})
}

// Internal 500.
func Internal(w http.ResponseWriter, msg string) {
	writeError(w, apiError{HTTP: http.StatusInternalServerError, Code: "INTERNAL", Detail: msg})
}
