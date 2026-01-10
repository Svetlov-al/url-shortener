package redirect

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"
)

//go:generate go run github.com/vektra/mockery/v2@latest  --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"
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

		url, err := urlGetter.GetURL(alias)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrURLNotFound):
				log.Info("url not found", slog.String("alias", alias))
				render.JSON(w, r, resp.Error("url not found"))
			default:
				log.Error("failed to get url", sl.Err(err))
				render.JSON(w, r, resp.Error("failed to get url"))
			}
			return
		}

		log.Info("url found", slog.String("url", url))
		http.Redirect(w, r, url, http.StatusFound)
	}
}
