package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/heldtogether/traintrack/internal"
	"github.com/heldtogether/traintrack/internal/auth"
)

type Key string

const (
	CtxKeyUser Key = "user"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(&internal.Error{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
				Reason:  "missing or invalid Authorization header",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse or validate the token here (JWT or opaque token)
		userInfo, err := auth.VerifyToken(r.Context(), tokenString)
		if err != nil {
			log.Printf("invalid token: %s\n", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(&internal.Error{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
				Reason:  fmt.Sprintf("invalid token: %s", err.Error()),
			})
			return
		}

		// Store user info in context
		ctx := context.WithValue(r.Context(), CtxKeyUser, userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
