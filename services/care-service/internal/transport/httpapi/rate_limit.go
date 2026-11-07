package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var rateLimitRejections = promauto.NewCounter(prometheus.CounterOpts{
	Name: "care_http_rate_limit_rejections_total",
	Help: "Total authenticated Care Service requests rejected by the service-side actor rate limit.",
})

type actorBucket struct {
	tokens   float64
	refilled time.Time
	lastSeen time.Time
}

// ActorRateLimiter is an in-memory per-instance safeguard. The API Gateway remains
// responsible for the distributed/public-edge rate limit across all replicas.
type ActorRateLimiter struct {
	mu          sync.Mutex
	rate        float64
	burst       float64
	buckets     map[uuid.UUID]*actorBucket
	now         func() time.Time
	idleTTL     time.Duration
	lastCleanup time.Time
}

func NewActorRateLimiter(requestsPerSecond, burst int) *ActorRateLimiter {
	if requestsPerSecond < 1 {
		requestsPerSecond = 20
	}
	if burst < 1 {
		burst = requestsPerSecond * 2
	}
	now := time.Now()
	return &ActorRateLimiter{
		rate: float64(requestsPerSecond), burst: float64(burst), buckets: make(map[uuid.UUID]*actorBucket),
		now: time.Now, idleTTL: 10 * time.Minute, lastCleanup: now,
	}
}

func (l *ActorRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := AuthenticatedActor(r.Context())
		if !ok {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
			return
		}
		allowed, retryAfter := l.allow(actor.ID)
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			rateLimitRejections.Inc()
			writeError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests", map[string]any{
				"retry_after_seconds": seconds,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *ActorRateLimiter) allow(actorID uuid.UUID) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastCleanup) >= l.idleTTL {
		for id, bucket := range l.buckets {
			if now.Sub(bucket.lastSeen) >= l.idleTTL {
				delete(l.buckets, id)
			}
		}
		l.lastCleanup = now
	}

	bucket, exists := l.buckets[actorID]
	if !exists {
		l.buckets[actorID] = &actorBucket{tokens: l.burst - 1, refilled: now, lastSeen: now}
		return true, 0
	}

	elapsed := now.Sub(bucket.refilled).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(l.burst, bucket.tokens+elapsed*l.rate)
		bucket.refilled = now
	}
	bucket.lastSeen = now
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}
	return false, time.Duration(math.Ceil((1 - bucket.tokens) / l.rate * float64(time.Second)))
}
