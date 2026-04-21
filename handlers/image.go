package handlers

import (
	"net/http"
)

func UploadImage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Imagen subida"))
}