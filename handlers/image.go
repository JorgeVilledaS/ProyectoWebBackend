package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Extraer el ID de la serie del path /series/42/image
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

	// Límite de 1MB
	r.ParseMultipartForm(1 << 20)
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image field is required")
		return
	}
	defer file.Close()

	// Validar extensión
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		writeError(w, http.StatusBadRequest, "only jpg, jpeg, png, webp, gif are allowed")
		return
	}

	// Crear carpeta uploads si no existe
	os.MkdirAll("uploads", 0755)

	// Nombre único para evitar colisiones
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

	// Guardar la URL en la base de datos
	imageURL := "/uploads/" + filename
	if _, err := h.db.Exec("UPDATE series SET image_url=$1 WHERE id=$2", imageURL, id); err != nil {
		writeError(w, http.StatusInternalServerError, "error updating image url")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"image_url": imageURL})
}