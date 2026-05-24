package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ebenamoafo2/workout-tracking/internal/store"
	"github.com/ebenamoafo2/workout-tracking/internal/tokens"
	"github.com/ebenamoafo2/workout-tracking/internal/utils"
)

// UserMiddleware holds a reference to the user store for token lookups
type UserMiddleware struct {
	UserStore store.UserStore
}

// contextKey prevents key collisions when storing values in request context
type contextKey string

// UserContextKey is the key for storing the authenticated user in context
const UserContextKey = contextKey("user")

// SetUser stores the given user in the request context
func SetUser(r *http.Request, user *store.User) *http.Request {
	ctx := context.WithValue(r.Context(), UserContextKey, user)
	return r.WithContext(ctx)
}

// GetUser retrieves the user from context. Panics if Authenticate was not run first.
func GetUser(r *http.Request) *store.User {
	user, ok := r.Context().Value(UserContextKey).(*store.User)
	if !ok {
		panic("missing user in request context")
	}
	return user
}

// Authenticate validates the Bearer token and attaches the user to context.
// Falls through as anonymous if no token is provided.
func (um *UserMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		// extract the token from the Authorization header
		authHeader := r.Header.Get("Authorization")

		// no token — continue as anonymous
		if authHeader == "" {
			r = SetUser(r, store.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// expect: Bearer <token>
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid authorization header"})
			return
		}

		token := headerParts[1]

		// look up the user tied to this token
		user, err := um.UserStore.GetUserToken(tokens.ScopeAuth, token)
		if err != nil {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid token"})
			return
		}

		// token valid but expired or revoked
		if user == nil {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "token expired or invalid"})
			return
		}

		// token valid — attach user to context and continue
		r = SetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// RequireUser blocks anonymous users. Must run after Authenticate.
func (um *UserMiddleware) RequireUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user from context (set by Authenticate)
		user := GetUser(r)

		// If the user is anonymous, block access
		if user.IsAnonymous() {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "you must be logged in to access this route"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
