package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/omaklabs/base/internal/auth"
	"github.com/omaklabs/base/internal/db"
)

// Session returns middleware that reads the session_token cookie from the
// incoming request, validates it via auth.ValidateSession, and — if valid —
// fetches the corresponding user from the database and stores it in the request
// context. Downstream handlers retrieve the user with CurrentUser.
func Session(sqlDB *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_token")
			if err == nil && cookie.Value != "" {
				q := db.New(sqlDB)
				userID, err := auth.ValidateSession(r.Context(), q, cookie.Value)
				if err == nil {
					user, err := q.GetUserByID(r.Context(), userID)
					if err == nil {
						ctx := context.WithValue(r.Context(), userKey, &user)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CurrentUser extracts the authenticated user from the request context.
// Returns nil when the request is unauthenticated.
func CurrentUser(r *http.Request) *db.User {
	user, _ := r.Context().Value(userKey).(*db.User)
	return user
}

// RequireAuth is middleware that redirects unauthenticated requests to /login.
// It should be applied after Session middleware on routes that require a logged-
// in user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentUser(r) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithUser returns a copy of r whose context carries the given user. This is
// exported so that tests can inject an authenticated user without setting up a
// full session cookie flow.
func WithUser(r *http.Request, user *db.User) *http.Request {
	ctx := context.WithValue(r.Context(), userKey, user)
	return r.WithContext(ctx)
}
