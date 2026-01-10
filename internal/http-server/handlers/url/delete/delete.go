package delete

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "url-shortener/internal/lib/api/response"
)

type URLDeleter interface {
	DeleteURL(alias string) error
}

//go:generate go run github.com/vektra/mockery/v2@latest --name=URLDeleter
func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty", slog.String("alias", alias))
			render.JSON(w, r, resp.Error("alias is empty"))
			return
		}

		if err := urlDeleter.DeleteURL(alias); err != nil {
			render.JSON(w, r, resp.Error("failed to delete url"))
			return
		}

		log.Info("url deleted", slog.String("alias", alias))
		w.WriteHeader(http.StatusNoContent)
	}
}
