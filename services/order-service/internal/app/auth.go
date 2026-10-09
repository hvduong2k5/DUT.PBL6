package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"strings"
	"time"
)

type Claims struct {
	Subject   string `json:"sub"`
	Role      string `json:"role"`
	Warehouse string `json:"warehouse,omitempty"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Expires   int64  `json:"exp"`
	Issued    int64  `json:"iat"`
}

func SignToken(secret string, c Claims) string {
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	b, _ := json.Marshal(c)
	body := head + "." + base64.RawURLEncoding.EncodeToString(b)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s *Service) Authenticate(token string) (domain.Principal, error) {
	if len(token) > 8192 {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	parts := strings.Split(strings.TrimPrefix(token, "Bearer "), ".")
	if len(parts) != 3 {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	header, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	var hd struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(header, &hd) != nil || hd.Alg != "HS256" {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	h := hmac.New(sha256.New, []byte(s.C.JWTSecret))
	h.Write([]byte(parts[0] + "." + parts[1]))
	signature, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil || !hmac.Equal(signature, h.Sum(nil)) {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	payload, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	var c Claims
	if json.Unmarshal(payload, &c) != nil || c.Issuer != "order-sandbox" || c.Audience != "order-api" || c.Expires <= time.Now().Unix() || c.Issued > time.Now().Add(time.Minute).Unix() {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	if _, e = uuid.Parse(c.Subject); e != nil {
		return domain.Principal{}, domain.E("UNAUTHENTICATED", 401)
	}
	p := domain.Principal{ID: c.Subject, Role: c.Role, Warehouse: c.Warehouse}
	if p.Role != "CUSTOMER" && p.Role != "GUEST" && !p.Staff() {
		return p, domain.E("UNAUTHENTICATED", 401)
	}
	return p, nil
}
func (s *Service) Internal(token string) bool {
	return len(token) > 0 && hmac.Equal([]byte(token), []byte(s.C.InternalToken))
}
func (s *Service) Cursor(o domain.Order, scope, filter string) string {
	body := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%s|%s|%s", scope, filter, o.CreatedAt.Format(time.RFC3339Nano), o.ID)))
	mac := hmac.New(sha256.New, []byte(s.C.JWTSecret))
	mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Service) decodeCursor(value, scope, filter string, at *time.Time, id *string) error {
	p := strings.Split(value, ".")
	if len(p) != 2 {
		return domain.E("INVALID_CURSOR", 400)
	}
	sig, e := base64.RawURLEncoding.DecodeString(p[1])
	if e != nil {
		return domain.E("INVALID_CURSOR", 400)
	}
	h := hmac.New(sha256.New, []byte(s.C.JWTSecret))
	h.Write([]byte(p[0]))
	if !hmac.Equal(sig, h.Sum(nil)) {
		return domain.E("INVALID_CURSOR", 400)
	}
	b, e := base64.RawURLEncoding.DecodeString(p[0])
	if e != nil {
		return domain.E("INVALID_CURSOR", 400)
	}
	fields := strings.Split(string(b), "|")
	if len(fields) != 4 || fields[0] != scope || fields[1] != filter {
		return domain.E("INVALID_CURSOR", 400)
	}
	*at, e = time.Parse(time.RFC3339Nano, fields[2])
	if e != nil {
		return domain.E("INVALID_CURSOR", 400)
	}
	*id = fields[3]
	if _, e = uuid.Parse(*id); e != nil {
		return domain.E("INVALID_CURSOR", 400)
	}
	return nil
}
