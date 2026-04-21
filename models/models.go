package models

// Series representa una serie en el tracker.
type Series struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	CurrentEpisode int     `json:"current_episode"`
	TotalEpisodes  int     `json:"total_episodes"`
	ImageURL       string  `json:"image_url"`
	AverageRating  float64 `json:"average_rating"`
	RatingCount    int     `json:"rating_count"`
}

// CreateSeriesInput es el payload esperado al crear una serie.
type CreateSeriesInput struct {
	Name           string `json:"name"`
	CurrentEpisode int    `json:"current_episode"`
	TotalEpisodes  int    `json:"total_episodes"`
}

// UpdateSeriesInput es el payload esperado al editar una serie.
type UpdateSeriesInput struct {
	Name           *string `json:"name"`
	CurrentEpisode *int    `json:"current_episode"`
	TotalEpisodes  *int    `json:"total_episodes"`
}

// Rating representa un rating de una serie.
type Rating struct {
	ID       int    `json:"id"`
	SeriesID int    `json:"series_id"`
	Score    int    `json:"score"`
	Comment  string `json:"comment"`
}

// CreateRatingInput es el payload esperado al crear un rating.
type CreateRatingInput struct {
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

// PaginatedSeries es la respuesta paginada de series.
type PaginatedSeries struct {
	Data       []Series `json:"data"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	TotalPages int      `json:"total_pages"`
}

// ErrorResponse es la estructura estándar de error.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}