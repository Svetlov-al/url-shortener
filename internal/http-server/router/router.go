package router

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/middleware/auth"
	mwlogger "url-shortener/internal/http-server/middleware/logger"
)

type Storage interface {
	save.URLSaver
	redirect.URLGetter
	delete.URLDeleter
}

func New(
	log *slog.Logger,
	storage Storage,
	adminChecker auth.AdminChecker,
	appSecret string,
	ssoTimeout time.Duration,
) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(mwlogger.New(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)

	r.Route("/url", func(r chi.Router) {
		r.Use(auth.AdminOnly(log, adminChecker, appSecret, ssoTimeout))
		r.Post("/", save.New(log, storage))
		r.Delete("/{alias}", delete.New(log, storage))
	})

	r.Get("/{alias}", redirect.New(log, storage))

	return r
}
