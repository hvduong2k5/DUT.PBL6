package transport

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

var quotaScript = redis.NewScript(`local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('PEXPIRE',KEYS[1],ARGV[1]) end; return n`)

func guestClientIP(r *http.Request, trusted []string) string {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	isTrusted := func(ip string) bool {
		for _, cidr := range trusted {
			_, block, err := net.ParseCIDR(cidr)
			if err == nil && block.Contains(net.ParseIP(ip)) {
				return true
			}
		}
		return false
	}
	// Walk from the direct peer inward. A client cannot pick its quota bucket
	// by prepending a forged forwarded address ahead of the real untrusted IP.
	chain := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(chain) - 1; i >= 0 && isTrusted(peer); i-- {
		next := strings.TrimSpace(chain[i])
		if net.ParseIP(next) == nil {
			break
		}
		peer = next
	}
	return peer
}

func guestQuota(s *app.Service, r *http.Request, key string, limit int, window time.Duration) error {
	if s.Redis == nil {
		return domain.E("GUEST_LIMIT_UNAVAILABLE", 503)
	}
	n, e := quotaScript.Run(r.Context(), s.Redis, []string{key}, window.Milliseconds()).Int()
	if e != nil {
		return domain.E("GUEST_LIMIT_UNAVAILABLE", 503)
	}
	if n > limit {
		return domain.E("GUEST_RATE_LIMITED", 429)
	}
	return nil
}

func guestFailure(s *app.Service, w http.ResponseWriter, err error) {
	code, status := s.ErrorCode(err)
	requestID := uuid.NewString()
	w.Header().Set("X-Request-ID", requestID)
	write(w, status, map[string]any{"code": code, "message": "The guest request could not be completed.", "requestId": requestID})
}

func issueGuestSession(s *app.Service, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.C.AuthMode == "keycloak" && !s.C.GuestSessionIssuanceEnabled {
		guestFailure(s, w, domain.E("GUEST_SESSIONS_DISABLED", 403))
		return
	}
	if s.C.AuthMode == "keycloak" {
		var in *struct{}
		if r.ContentLength != 0 {
			if e := decode(w, r, &in); e != nil {
				guestFailure(s, w, e)
				return
			}
			if in == nil {
				guestFailure(s, w, domain.E("INVALID_REQUEST", 400))
				return
			}
		}
		if s.GuestSessions() == nil {
			guestFailure(s, w, domain.E("GUEST_SESSION_UNAVAILABLE", 503))
			return
		}
		ip := guestClientIP(r, s.C.GuestTrustedProxyCIDRs)
		limit := s.C.GuestIssuancePerHour
		if limit < 1 {
			limit = 10
		}
		if e := guestQuota(s, r, fmt.Sprintf("guest-issue:%x", sha256.Sum256([]byte(ip))), limit, time.Hour); e != nil {
			w.Header().Set("Retry-After", "3600")
			guestFailure(s, w, e)
			return
		}
	}
	id, now := uuid.NewString(), time.Now()
	expires := now.Add(30 * 24 * time.Hour)
	if s.GuestSessions() != nil || s.C.AuthMode == "keycloak" {
		expires = now.Add(24 * time.Hour)
	}
	token := app.SignToken(s.C.JWTSecret, app.Claims{Subject: id, Role: "GUEST", Issuer: "order-sandbox", Audience: "order-api", Issued: now.Unix(), Expires: expires.Unix()})
	if s.C.AuthMode == "keycloak" {
		var e error
		token, e = s.SignGuestSession(id, now, expires)
		if e != nil {
			guestFailure(s, w, e)
			return
		}
	}
	if s.GuestSessions() != nil {
		if e := s.GuestSessions().CreateGuestSession(r.Context(), id, token, now, expires); e != nil {
			guestFailure(s, w, domain.E("GUEST_SESSION_UNAVAILABLE", 503))
			return
		}
	}
	write(w, 201, map[string]any{"session_id": id, "access_token": token, "expires_at": expires.UTC()})
}

func authorizeGuestRequest(s *app.Service, p domain.Principal, r *http.Request) error {
	if s.C.AuthMode != "keycloak" {
		return nil
	} // Preserve sandbox route behavior.
	if e := app.AuthorizeRoute(p, r.Method, r.URL.Path); e != nil {
		return e
	}
	if p.PrincipalKind() != domain.Guest || s.C.AuthMode != "keycloak" {
		return nil
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/cart") && r.Method != "GET" {
		if !s.C.GuestCartMutationEnabled {
			return domain.E("GUEST_CART_DISABLED", 403)
		}
		return guestQuota(s, r, "guest-cart-rate:"+p.ID, 60, time.Minute)
	}
	return nil
}
