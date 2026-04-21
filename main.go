package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"series-tracker/db"
	"series-tracker/handlers"
	"series-tracker/middleware"
)

func main() {
	// Conectar a PostgreSQL
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Crear tablas si no existen
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Crear carpeta de uploads para imágenes
	os.MkdirAll("uploads", 0755)

	// Inicializar handlers con la DB inyectada
	h := handlers.New(database)

	mux := http.NewServeMux()

	// ─── Rutas ────────────────────────────────────────────────────────────────

	// GET /series             — listar series (con paginación, búsqueda, orden)
	// POST /series            — crear una nueva serie
	mux.HandleFunc("/series", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListSeries(w, r)
		case http.MethodPost:
			h.CreateSeries(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// /series/* — rutas con ID o sub-recursos
	mux.HandleFunc("/series/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Normalizar: quitar trailing slash
		path = strings.TrimRight(path, "/")

		// Detectar sub-ruta /series/:id/image
		if strings.HasSuffix(path, "/image") && r.Method == http.MethodPost {
			h.UploadImage(w, r)
			return
		}

		// Detectar sub-ruta /series/:id/rating
		if strings.HasSuffix(path, "/rating") {
			switch r.Method {
			case http.MethodGet:
				h.GetRatings(w, r)
			case http.MethodPost:
				h.CreateRating(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// Rutas /series/:id
		switch r.Method {
		case http.MethodGet:
			h.GetSeries(w, r)
		case http.MethodPut:
			h.UpdateSeries(w, r)
		case http.MethodDelete:
			h.DeleteSeries(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// DELETE /ratings/:id — eliminar un rating específico
	mux.HandleFunc("/ratings/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			h.DeleteRating(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Servir imágenes subidas como archivos estáticos
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Aplicar middleware CORS a todas las rutas
	handler := middleware.CORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎬 Series Tracker API running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}