package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"
	"unicode"

	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
)

// maxReplayBody bounds what is stored for replay. A larger response abandons the
// key instead; the retry then fails safely on the optimistic version or slug.
const maxReplayBody = 4 << 20

// replayHeaders are the response headers restored on replay.
var replayHeaders = []string{"Content-Type", "Location", "ETag", "Cache-Control"}

type capture struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	over   bool
}

func (c *capture) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
	c.ResponseWriter.WriteHeader(code)
}
func (c *capture) Write(p []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	if c.body.Len()+len(p) > maxReplayBody {
		c.over = true
	} else {
		c.body.Write(p)
	}
	return c.ResponseWriter.Write(p)
}

func validKey(k string) bool {
	if k == "" || len(k) > 255 {
		return false
	}
	for _, r := range k {
		if r > unicode.MaxASCII || !unicode.IsPrint(r) || r == ' ' {
			return false
		}
	}
	return true
}

// Idempotency makes retried POSTs safe: the same key and payload from the same
// principal returns the first response, a different payload is a 409. A request
// without the header is not deduplicated. It must run after Authenticate.
func Idempotency(store domain.IdempotencyStore, maxBody int64, fail func(http.ResponseWriter, *http.Request, int, string, string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if store == nil || r.Method != http.MethodPost || key == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !validKey(key) {
				fail(w, r, 400, "BAD_REQUEST", "Idempotency-Key must be 1..255 printable ASCII characters")
				return
			}
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
			if err != nil {
				fail(w, r, 400, "BAD_REQUEST", "invalid request body")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			sum := sha256.Sum256(append([]byte(r.Method+"\n"+r.URL.Path+"\n"), body...))
			hash := hex.EncodeToString(sum[:])
			principal := Identity(r.Context()).ID
			rec, created, err := store.Begin(r.Context(), principal, key, hash)
			if err != nil {
				fail(w, r, 500, "INTERNAL_ERROR", "internal server error")
				return
			}
			if !created {
				switch {
				case rec.RequestHash != hash:
					fail(w, r, 409, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used with a different request")
				case !rec.Completed:
					w.Header().Set("Retry-After", "1")
					fail(w, r, 409, "IDEMPOTENCY_IN_PROGRESS", "a request with this Idempotency-Key is still being processed")
				default:
					for name, value := range rec.Headers {
						w.Header().Set(name, value)
					}
					w.Header().Set("Idempotent-Replayed", "true")
					w.WriteHeader(rec.Status)
					_, _ = w.Write(rec.Body)
				}
				return
			}
			c := &capture{ResponseWriter: w}
			// A panic must not leave the key stuck IN_PROGRESS for the stale window.
			finished := false
			defer func() {
				if finished {
					return
				}
				ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Second)
				defer cancel()
				_ = store.Abandon(ctx, principal, key)
			}()
			next.ServeHTTP(c, r)
			finished = true
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Second)
			defer cancel()
			if c.status == 0 || c.status >= 500 || c.over {
				_ = store.Abandon(ctx, principal, key)
				return
			}
			headers := map[string]string{}
			for _, name := range replayHeaders {
				if v := w.Header().Get(name); v != "" {
					headers[name] = v
				}
			}
			_ = store.Complete(ctx, principal, key, c.status, headers, c.body.Bytes())
		})
	}
}
