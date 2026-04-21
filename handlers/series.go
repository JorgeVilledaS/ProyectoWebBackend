package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"series-tracker/models"
)

// Handler contiene la dependencia de la base de datos.
type Handler struct {
	db *sql.DB
}

// New crea un nuevo Handler con la DB inyectada.
func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// ---------- Helpers ----------

// writeJSON serializa v como JSON y lo escribe con el status code dado.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError escribe un error estándar en JSON.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}

// writeValidationError escribe errores de validación con campo→mensaje.
func writeValidationError(w http.ResponseWriter, details map[string]string) {
	writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
		Error:   "validation_error",
		Details: details,
	})
}

// extractID obtiene el segmento de ID de una URL como /series/42 o /series/42/rating.
// segmentIndex indica en qué posición del path (split por /) está el ID.
func extractID(path string, segmentIndex int) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) <= segmentIndex {
		return 0, fmt.Errorf("missing id")
	}
	return strconv.Atoi(parts[segmentIndex])
}

// ---------- Series Handlers ----------

// ListSeries maneja GET /series
// Soporta: ?page=, ?limit=, ?q= (búsqueda), ?sort=, ?order=
func (h *Handler) ListSeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parámetros de paginación
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Búsqueda por nombre
	search := strings.TrimSpace(q.Get("q"))

	// Ordenamiento: columnas permitidas para evitar SQL injection
	allowedSorts := map[string]bool{
		"name": true, "current_episode": true,
		"total_episodes": true, "id": true, "created_at": true,
	}
	sortBy := q.Get("sort")
	if !allowedSorts[sortBy] {
		sortBy = "id"
	}
	order := strings.ToUpper(q.Get("order"))
	if order != "ASC" && order != "DESC" {
		order = "ASC"
	}

	// Construir WHERE dinámico
	args := []any{}
	where := ""
	if search != "" {
		args = append(args, "%"+search+"%")
		where = "WHERE s.name ILIKE $1"
	}

	// Contar total de registros
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM series s %s
	`, where)
	var total int
	if err := h.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "error counting series")
		return
	}

	// Agregar parámetros de paginación a los args
	args = append(args, limit, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(args)-1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args))

	// Query principal con JOIN para calcular average_rating
	dataQuery := fmt.Sprintf(`
		SELECT
			s.id, s.name, s.current_episode, s.total_episodes, s.image_url,
			COALESCE(AVG(r.score), 0) AS average_rating,
			COUNT(r.id) AS rating_count
		FROM series s
		LEFT JOIN ratings r ON r.series_id = s.id
		%s
		GROUP BY s.id
		ORDER BY s.%s %s
		LIMIT %s OFFSET %s
	`, where, sortBy, order, limitPlaceholder, offsetPlaceholder)

	rows, err := h.db.Query(dataQuery, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error querying series")
		return
	}
	defer rows.Close()

	series := []models.Series{}
	for rows.Next() {
		var s models.Series
		if err := rows.Scan(&s.ID, &s.Name, &s.CurrentEpisode, &s.TotalEpisodes,
			&s.ImageURL, &s.AverageRating, &s.RatingCount); err != nil {
			continue
		}
		series = append(series, s)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	writeJSON(w, http.StatusOK, models.PaginatedSeries{
		Data:       series,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// GetSeries maneja GET /series/:id
func (h *Handler) GetSeries(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	var s models.Series
	err = h.db.QueryRow(`
		SELECT
			s.id, s.name, s.current_episode, s.total_episodes, s.image_url,
			COALESCE(AVG(r.score), 0), COUNT(r.id)
		FROM series s
		LEFT JOIN ratings r ON r.series_id = s.id
		WHERE s.id = $1
		GROUP BY s.id
	`, id).Scan(&s.ID, &s.Name, &s.CurrentEpisode, &s.TotalEpisodes,
		&s.ImageURL, &s.AverageRating, &s.RatingCount)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching series")
		return
	}

	writeJSON(w, http.StatusOK, s)
}

// CreateSeries maneja POST /series
func (h *Handler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	var input models.CreateSeriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validación server-side
	errs := map[string]string{}
	if strings.TrimSpace(input.Name) == "" {
		errs["name"] = "name is required"
	}
	if input.TotalEpisodes <= 0 {
		errs["total_episodes"] = "total_episodes must be greater than 0"
	}
	if input.CurrentEpisode < 0 {
		errs["current_episode"] = "current_episode cannot be negative"
	}
	if input.CurrentEpisode > input.TotalEpisodes && input.TotalEpisodes > 0 {
		errs["current_episode"] = "current_episode cannot exceed total_episodes"
	}
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	var id int
	err := h.db.QueryRow(`
		INSERT INTO series (name, current_episode, total_episodes)
		VALUES ($1, $2, $3)
		RETURNING id
	`, strings.TrimSpace(input.Name), input.CurrentEpisode, input.TotalEpisodes).Scan(&id)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "error creating series")
		return
	}

	// Devolver la serie recién creada — 201 Created
	h.GetSeriesByID(w, id, http.StatusCreated)
}

// GetSeriesByID es un helper interno para obtener y devolver una serie.
func (h *Handler) GetSeriesByID(w http.ResponseWriter, id int, status int) {
	var s models.Series
	err := h.db.QueryRow(`
		SELECT
			s.id, s.name, s.current_episode, s.total_episodes, s.image_url,
			COALESCE(AVG(r.score), 0), COUNT(r.id)
		FROM series s
		LEFT JOIN ratings r ON r.series_id = s.id
		WHERE s.id = $1
		GROUP BY s.id
	`, id).Scan(&s.ID, &s.Name, &s.CurrentEpisode, &s.TotalEpisodes,
		&s.ImageURL, &s.AverageRating, &s.RatingCount)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching series")
		return
	}
	writeJSON(w, status, s)
}

// UpdateSeries maneja PUT /series/:id
func (h *Handler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	// Verificar que existe
	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", id).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	var input models.UpdateSeriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validación de los campos presentes
	errs := map[string]string{}
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		errs["name"] = "name cannot be empty"
	}
	if input.TotalEpisodes != nil && *input.TotalEpisodes <= 0 {
		errs["total_episodes"] = "total_episodes must be greater than 0"
	}
	if input.CurrentEpisode != nil && *input.CurrentEpisode < 0 {
		errs["current_episode"] = "current_episode cannot be negative"
	}
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	// Construir UPDATE dinámico solo con los campos enviados
	sets := []string{}
	args := []any{}
	argIdx := 1

	if input.Name != nil {
		sets = append(sets, fmt.Sprintf("name=$%d", argIdx))
		args = append(args, strings.TrimSpace(*input.Name))
		argIdx++
	}
	if input.CurrentEpisode != nil {
		sets = append(sets, fmt.Sprintf("current_episode=$%d", argIdx))
		args = append(args, *input.CurrentEpisode)
		argIdx++
	}
	if input.TotalEpisodes != nil {
		sets = append(sets, fmt.Sprintf("total_episodes=$%d", argIdx))
		args = append(args, *input.TotalEpisodes)
		argIdx++
	}

	if len(sets) == 0 {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE series SET %s WHERE id=$%d", strings.Join(sets, ", "), argIdx)
	if _, err := h.db.Exec(query, args...); err != nil {
		writeError(w, http.StatusInternalServerError, "error updating series")
		return
	}

	h.GetSeriesByID(w, id, http.StatusOK)
}

// DeleteSeries maneja DELETE /series/:id
func (h *Handler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	result, err := h.db.Exec("DELETE FROM series WHERE id=$1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error deleting series")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	// 204 No Content — eliminado correctamente, sin body
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Image Upload ----------

// UploadImage maneja POST /series/:id/image
// Acepta multipart/form-data con campo "image". Guarda el archivo en ./uploads/.
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	// Verificar que la serie existe
	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", id).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	// Límite de 1MB por imagen
	r.ParseMultipartForm(1 << 20)
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image field is required")
		return
	}
	defer file.Close()

	// Validar tipo de archivo por extensión
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		writeError(w, http.StatusBadRequest, "only jpg, jpeg, png, webp, gif are allowed")
		return
	}

	// Guardar con nombre único basado en timestamp
	filename := fmt.Sprintf("%d_%d%s", id, time.Now().UnixMilli(), ext)
	uploadPath := filepath.Join("uploads", filename)

	dst, err := os.Create(uploadPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error saving image")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "error writing image")
		return
	}

	// Guardar URL relativa en la base de datos
	imageURL := "/uploads/" + filename
	if _, err := h.db.Exec("UPDATE series SET image_url=$1 WHERE id=$2", imageURL, id); err != nil {
		writeError(w, http.StatusInternalServerError, "error updating image url")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"image_url": imageURL})
}