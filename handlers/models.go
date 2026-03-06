package handlers

import (
	"net/http"

	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/ratelimit"
)

type Server struct {
	Mux       *http.ServeMux
	RateLimit *ratelimit.RateLimiter
}

type ApiConfig struct {
	Db   *database.Queries
	Serv Server
}
