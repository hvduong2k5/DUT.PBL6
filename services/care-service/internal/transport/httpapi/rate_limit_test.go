package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestActorRateLimiterRejectsOnlyActorOverBurstAndRecovers(t *testing.T) {
	clock := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	limiter := NewActorRateLimiter(2, 2)
	limiter.now = func() time.Time { return clock }
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	actorA := uuid.New()
	actorB := uuid.New()

	request := func(actorID uuid.UUID) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", nil)
		ctx := context.WithValue(req.Context(), actorContextKey{}, Actor{ID: actorID})
		handler.ServeHTTP(recorder, req.WithContext(ctx))
		return recorder
	}

	require.Equal(t, http.StatusNoContent, request(actorA).Code)
	require.Equal(t, http.StatusNoContent, request(actorA).Code)
	rejected := request(actorA)
	require.Equal(t, http.StatusTooManyRequests, rejected.Code)
	require.Equal(t, "1", rejected.Header().Get("Retry-After"))
	require.Contains(t, rejected.Body.String(), `"code":"RATE_LIMITED"`)
	require.Contains(t, rejected.Body.String(), `"retry_after_seconds":1`)

	require.Equal(t, http.StatusNoContent, request(actorB).Code, "one actor must not consume another actor's bucket")
	clock = clock.Add(500 * time.Millisecond)
	require.Equal(t, http.StatusNoContent, request(actorA).Code, "tokens must refill at the configured rate")
}

func TestActorRateLimiterRequiresAuthenticatedActor(t *testing.T) {
	limiter := NewActorRateLimiter(1, 1)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)

	limiter.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not run")
	})).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestActorRateLimiterRemovesIdleActorBuckets(t *testing.T) {
	clock := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	limiter := NewActorRateLimiter(1, 1)
	limiter.now = func() time.Time { return clock }
	limiter.lastCleanup = clock
	idleActor := uuid.New()
	activeActor := uuid.New()

	allowed, _ := limiter.allow(idleActor)
	require.True(t, allowed)
	require.Len(t, limiter.buckets, 1)

	clock = clock.Add(limiter.idleTTL)
	allowed, _ = limiter.allow(activeActor)
	require.True(t, allowed)
	require.NotContains(t, limiter.buckets, idleActor)
	require.Contains(t, limiter.buckets, activeActor)
}
