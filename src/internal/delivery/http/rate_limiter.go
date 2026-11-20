package http

import (
	"encoding/json"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// stores rate limiters for user ID
var userLimiters sync.Map

func RateLimitMiddleware(reqPerUser float64, burst int) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract userID from context
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "user identification required for rate limiting"})
				return
			}

			// get or create rate limiter for user
			limiterInterface, _ := userLimiters.LoadOrStore(userID, rate.NewLimiter(rate.Limit(reqPerUser), burst))
			limiter := limiterInterface.(*rate.Limiter)

			// Check if request is allowed
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "rate limit exceeded, please try again later"})
				return
			}

			// Request is allowed
			next(w, r)
		}
	}
}
