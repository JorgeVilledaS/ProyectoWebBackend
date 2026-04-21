package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"series-tracker/models"
)

// ---------- Rating Handlers ----------

// CreateRating maneja POST /series/:id/rating
// Recibe score (1-10) y comment opcional.
func (h *Handler) CreateRating(w http.ResponseWriter, r *http.Request) {
	// Extraer el ID de la serie del path /series/42/rating
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	// Verificar que la serie existe antes de insertar
	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", id).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	var input models.CreateRatingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validar que el score esté en el rango permitido
	if input.Score < 1 || input.Score > 10 {
		writeValidationError(w, map[string]string{
			"score": "score must be between 1 and 10",
		})
		return
	}

	var ratingID int
	err = h.db.QueryRow(`
		INSERT INTO ratings (series_id, score, comment)
		VALUES ($1, $2, $3)
		RETURNING id
	`, id, input.Score, strings.TrimSpace(input.Comment)).Scan(&ratingID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "error saving rating")
		return
	}

	var rating models.Rating
	h.db.QueryRow("SELECT id, series_id, score, comment FROM ratings WHERE id=$1", ratingID).
		Scan(&rating.ID, &rating.SeriesID, &rating.Score, &rating.Comment)

	// 201 Created al crear un recurso nuevo
	writeJSON(w, http.StatusCreated, rating)
}

// GetRatings maneja GET /series/:id/rating
// Devuelve todos los ratings de una serie junto con la media calculada.
func (h *Handler) GetRatings(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", id).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	rows, err := h.db.Query(`
		SELECT id, series_id, score, comment
		FROM ratings
		WHERE series_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching ratings")
		return
	}
	defer rows.Close()

	ratings := []models.Rating{}
	for rows.Next() {
		var rt models.Rating
		if err := rows.Scan(&rt.ID, &rt.SeriesID, &rt.Score, &rt.Comment); err != nil {
			continue
		}
		ratings = append(ratings, rt)
	}

	// Calcular promedio en el servidor también
	var avg float64
	h.db.QueryRow("SELECT COALESCE(AVG(score), 0) FROM ratings WHERE series_id=$1", id).Scan(&avg)

	writeJSON(w, http.StatusOK, map[string]any{
		"ratings": ratings,
		"average": avg,
		"count":   len(ratings),
	})
}

// DeleteRating maneja DELETE /ratings/:id
func (h *Handler) DeleteRating(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rating id")
		return
	}

	result, err := h.db.Exec("DELETE FROM ratings WHERE id=$1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error deleting rating")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "rating not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}