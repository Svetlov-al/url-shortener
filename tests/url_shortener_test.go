package tests

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/router"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage/sqlite"
)

const (
	testAppSecret = "dev-secret"
)

func TestURLShortener_HappyPath(t *testing.T) {
	e, _ := newTestClient(t)

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithHeader("Authorization", "Bearer "+makeHS256JWT(t, testAppSecret, 1, time.Now().Add(10*time.Minute))).
		Expect().
		Status(200).
		JSON().Object().
		ContainsKey("alias")
}

//nolint:funlen
func TestURLShortener_SaveRedirectDelete(t *testing.T) {
	testCases := []struct {
		name  string
		url   string
		alias string
		error string
	}{
		{
			name:  "Valid URL",
			url:   gofakeit.URL(),
			alias: gofakeit.Word() + gofakeit.Word(),
		},
		{
			name:  "Invalid URL",
			url:   "invalid_url",
			alias: gofakeit.Word(),
			error: "поле URL должно быть валидным URL",
		},
		{
			name:  "Empty Alias",
			url:   gofakeit.URL(),
			alias: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e, baseURL := newTestClient(t)

			// Save

			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithHeader("Authorization", "Bearer "+makeHS256JWT(t, testAppSecret, 1, time.Now().Add(10*time.Minute))).
				Expect().Status(http.StatusOK).
				JSON().Object()

			if tc.error != "" {
				resp.NotContainsKey("alias")

				resp.Value("error").String().IsEqual(tc.error)

				return
			}

			alias := tc.alias

			if tc.alias != "" {
				resp.Value("alias").String().IsEqual(tc.alias)
			} else {
				resp.Value("alias").String().NotEmpty()

				alias = resp.Value("alias").String().Raw()
			}

			// Redirect

			testRedirect(t, baseURL, alias, tc.url)

			// Delete

			e.DELETE("/"+path.Join("url", alias)).
				WithHeader("Authorization", "Bearer "+makeHS256JWT(t, testAppSecret, 1, time.Now().Add(10*time.Minute))).
				Expect().Status(http.StatusNoContent)

			// Redirect

			testRedirectNotFound(t, baseURL, alias)
		})
	}
}

type fakeSSO struct{}

func (fakeSSO) IsAdmin(_ context.Context, userID int64) (bool, error) {
	return userID == 1, nil
}

func newTestClient(t *testing.T) (*httpexpect.Expect, string) {
	t.Helper()

	dbPath := tempDBPath(t)
	st, err := sqlite.New(dbPath)
	require.NoError(t, err)

	log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	r := router.New(log, st, fakeSSO{}, testAppSecret, 0)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	e := httpexpect.Default(t, srv.URL)
	e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Content-Type", "application/json")
	})
	return e, srv.URL
}

func tempDBPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "url-shortener-*.db")
	require.NoError(t, err)
	path := f.Name()
	require.NoError(t, f.Close())
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

func makeHS256JWT(t *testing.T, secret string, userID int64, exp time.Time) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     exp.Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func testRedirect(t *testing.T, baseURL string, alias string, urlToRedirect string) {
	u, err := url.Parse(baseURL)
	require.NoError(t, err)
	u.Path = alias

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)

	require.Equal(t, urlToRedirect, redirectedToURL)
}

func testRedirectNotFound(t *testing.T, baseURL string, alias string) {
	u, err := url.Parse(baseURL)
	require.NoError(t, err)
	u.Path = alias

	_, err = api.GetRedirect(u.String())
	require.ErrorIs(t, err, api.ErrInvalidStatusCode)
}
