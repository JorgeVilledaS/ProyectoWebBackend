package middleware

import "net/http"

// CORS agrega los headers necesarios para permitir peticiones cross-origin.
// CORS (Cross-Origin Resource Sharing) es una política del navegador que bloquea
// peticiones fetch() hacia orígenes distintos (otro puerto o dominio) a menos que
// el servidor lo permita explícitamente con estos headers.
// En este proyecto permitimos cualquier origen (*) para facilitar el desarrollo local.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Los preflight requests son OPTIONS: el navegador los manda antes del fetch real
		// para verificar que el servidor permite la operación.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}