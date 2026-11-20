package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockHandler is a HTTP handler used to verify the middleware
// correctly passes through requests that are allowed.
func mockHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// newAuthenticatedRequest builds a request with a user_id already set in the context simulating what AuthMiddleware would have done upstream.
func newAuthenticatedRequest(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestRateLimitMiddleware_AllowsUpToBurstThenBlocks(t *testing.T) {
	// rate.Limit(0) means the token bucket never refills, so only the
	// initial burst of 3 tokens is available. This makes the test deterministic without needing to wait on real time.
	middleware := RateLimitMiddleware(0, 3)
	handler := middleware(mockHandler)

	userID := "user-rate-limit-test"

	for i := 1; i <= 3; i++ {
		rec := httptest.NewRecorder()
		req := newAuthenticatedRequest(userID)

		handler(rec, req)

		assert.Equalf(t, http.StatusOK, rec.Code, "request %d should be allowed", i)
	}

	// 4th request should be rejected
	rec := httptest.NewRecorder()
	req := newAuthenticatedRequest(userID)

	handler(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "4th request should be rate limited")
}

func TestRateLimitMiddleware_MissingUserID_ReturnsUnauthorized(t *testing.T) {
	middleware := RateLimitMiddleware(0, 3)
	handler := middleware(mockHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil) // no user_id in context

	handler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRateLimitMiddleware_DifferentUsersHaveIndependentLimits(t *testing.T) {
	middleware := RateLimitMiddleware(0, 1)
	handler := middleware(mockHandler)

	// User A uses up their single allowed request.
	recA := httptest.NewRecorder()
	handler(recA, newAuthenticatedRequest("user-a-independent"))
	assert.Equal(t, http.StatusOK, recA.Code)

	recA2 := httptest.NewRecorder()
	handler(recA2, newAuthenticatedRequest("user-a-independent"))
	assert.Equal(t, http.StatusTooManyRequests, recA2.Code)

	// User B should still be allowed since limits are tracked per user.
	recB := httptest.NewRecorder()
	handler(recB, newAuthenticatedRequest("user-b-independent"))
	assert.Equal(t, http.StatusOK, recB.Code)
}
