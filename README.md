# Series Tracker — Backend

REST API de la app Tracker de Series, hecho con **Go** y **PostgreSQL**.

**Frontend repo:** https://github.com/JorgeVilledaS/series-tracker-frontend  
**Live app:** POR EL MOMENTO LOCAL

---

## Screenshot

PENDIENTE

---

## Requisitos

- Go 1.22+
- PostgreSQL 14+

---

## Correrlo Localmente

### 1. CLonar Repo

```bash
git clone https://github.com/JorgeVilledaS/series-tracker-backend
cd series-tracker-backend
```

### 2. Crear db

```sql
CREATE DATABASE series_tracker;
```

### 3. Configurar variables del env

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=series_tracker
PORT=8080
```

La aplicación se migra automáticamente.

### 4. Instalar dependencias y ejecutar.

```bash
go mod tidy
go run main.go
```

la API estará temporalmente en `http://localhost:8080`.

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /series | List all series (paginación, búsqueda, orden) |
| POST | /series | Create a series |
| GET | /series/:id | Get a series by ID |
| PUT | /series/:id | Update a series |
| DELETE | /series/:id | Delete a series |
| POST | /series/:id/image | Upload cover image |
| GET | /series/:id/rating | Get ratings for a series |
| POST | /series/:id/rating | Add a rating |
| DELETE | /ratings/:id | Delete a rating |

### Query params for GET /series

| Param | Example | Description |
|-------|---------|-------------|
| page | ?page=2 | Page number |
| limit | ?limit=5 | Results per page |
| q | ?q=break | Search by name |
| sort | ?sort=name | Sort field |
| order | ?order=DESC | Sort direction |

---

## CORS

CORS (Cross-Origin Resource Sharing) es una política de seguridad del navegador que bloquea las solicitudes fetch() hacia un origen distinto (incluye cambios de puerto).

Para permitir estas solicitudes, el servidor debe habilitarlas explícitamente mediante headers en la respuesta.

Este servidor envía los siguientes encabezados:

Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type

Para facilitarme el desarrollo permití que viniera de cualquier origen el acceso.

---

## Retos implementados

- Códigos HTTP correctos — 201 al crear, 204 al eliminar, 404 cuando no existe, 400 para entradas inválidas
- Validación en el servidor con respuestas de error en JSON por campo
- Paginación mediante ?page= y ?limit=
- Búsqueda con ?q= (insensible a mayúsculas/minúsculas, usando ILIKE)
- Ordenamiento con ?sort= y ?order=
- Subida de imágenes (multipart/form-data, límite de 1MB, almacenadas en /uploads)
- Sistema de calificaciones — tabla ratings separada con sus propios endpoints REST
- Especificación OpenAPI/Swagger (swagger.yaml)

---

## Reflection
Usar la librería estándar de go sin frameworks nos obliga a entender HTTP a mayor profundidad, pues cosas como ruteo, códigos de estado y encabezados los hacen "mágicamente" las herramientas más modernas. Postgres resultó fácil pues es la DB que usamos en el curso de DB1 y me pareció interesante usar el patrón COALESCE + LEFT JOIN para calcular promedios limpio y eficiente. 

Creo que si volvería usar estas tecnologías en proyectos de este mismo tamaño, pero en cosas más grandes me plantearía otras cosas. Aún así, la separación entre cliente y servidor (contrato REST + JSON) hizo que iterar en el frontend de forma independiente fuera realmente sencillo.
