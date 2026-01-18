package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
)

type AdminChecker interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type ctxKey string

const (
	ctxUserIDKey ctxKey = "user_id"
)

func UserIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(ctxUserIDKey)
	id, ok := v.(int64)
	return id, ok
}

func AdminOnly(
	log *slog.Logger,
	adminChecker AdminChecker,
	appSecret string,
	ssoTimeout time.Duration,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "middleware.auth.AdminOnly"

			log := log.With(
				slog.String("op", op),
			)

			token, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("unauthorized"))
				return
			}

			if appSecret == "" {
				log.Error("app secret is empty")
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("internal error"))
				return
			}

			claims, err := parseAndVerifyHS256JWT(token, []byte(appSecret))
			if err != nil {
				log.Info("invalid token", sl.Err(err))
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("unauthorized"))
				return
			}

			userID, ok := extractUserID(claims)
			if !ok || userID <= 0 {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("unauthorized"))
				return
			}

			ctx := r.Context()
			if ssoTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, ssoTimeout)
				defer cancel()
			}

			isAdmin, err := adminChecker.IsAdmin(ctx, userID)
			if err != nil {
				log.Error("failed to check admin status", sl.Err(err), slog.Int64("user_id", userID))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("internal error"))
				return
			}
			if !isAdmin {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Error("forbidden"))
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUserIDKey, userID)))
		})
	}
}

func bearerToken(authorization string) (string, error) {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return "", errors.New("authorization header is empty")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return "", errors.New("authorization header is not bearer")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	if token == "" {
		return "", errors.New("bearer token is empty")
	}
	return token, nil
}

func parseAndVerifyHS256JWT(tokenString string, secret []byte) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Ensure JSON numeric types are handled consistently.
	// (MapClaims stores decoded JSON values as interface{}.)
	if raw, ok := claims["exp"]; ok {
		switch raw.(type) {
		case json.Number, float64, int64, int32, int:
			// ok
		}
	}

	return claims, nil
}

func extractUserID(claims jwt.MapClaims) (int64, bool) {
	// Most common patterns.
	for _, key := range []string{"user_id", "uid", "userID", "userId"} {
		if v, ok := claims[key]; ok {
			id, ok := asInt64(v)
			if ok {
				return id, true
			}
		}
	}

	// JWT "sub" is often a string user id.
	if v, ok := claims["sub"]; ok {
		switch vv := v.(type) {
		case string:
			id, err := strconv.ParseInt(vv, 10, 64)
			if err == nil {
				return id, true
			}
		default:
			id, ok := asInt64(vv)
			if ok {
				return id, true
			}
		}
	}

	return 0, false
}

func asInt64(v any) (int64, bool) {
	switch vv := v.(type) {
	case int64:
		return vv, true
	case int32:
		return int64(vv), true
	case int:
		return int64(vv), true
	case float64:
		// JSON numbers decode as float64.
		if vv != float64(int64(vv)) {
			return 0, false
		}
		return int64(vv), true
	case json.Number:
		n, err := vv.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(vv, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
