package app

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	SourceTopic                                                                                                                  string
	ProfileToken                                                                                                                 string
	ConsumerGroup                                                                                                                string
	RedisDB                                                                                                                      int
	DB, Redis, Dependencies, Profile, InternalToken, JWTSecret, EncryptionKey, CallbackSecret, Receiver, Kafka, HTTP, GRPC, Mode string
	Poll                                                                                                                         time.Duration
	AutoComplete                                                                                                                 bool
}

func LoadConfig() (Config, error) {
	c := Config{ProfileToken: os.Getenv("PROFILE_INTERNAL_TOKEN"), DB: os.Getenv("DATABASE_URL"), Redis: os.Getenv("REDIS_ADDR"), Dependencies: os.Getenv("DEPENDENCY_URL"), Profile: os.Getenv("PROFILE_URL"), InternalToken: os.Getenv("ORDER_INTERNAL_TOKEN"), JWTSecret: os.Getenv("ORDER_JWT_SECRET"), EncryptionKey: os.Getenv("ORDER_ENCRYPTION_KEY"), CallbackSecret: os.Getenv("CALLBACK_SECRET"), Receiver: os.Getenv("PAYMENT_RECEIVER"), Kafka: os.Getenv("KAFKA_BROKERS"), HTTP: os.Getenv("HTTP_ADDR"), GRPC: os.Getenv("GRPC_ADDR"), Mode: os.Getenv("APP_MODE"), Poll: 100 * time.Millisecond, AutoComplete: os.Getenv("ENABLE_AUTO_COMPLETE") == "true"}
	if c.HTTP == "" {
		c.HTTP = ":8004"
	}
	if c.GRPC == "" {
		c.GRPC = ":9004"
	}
	if c.Mode != "sandbox" {
		return c, errors.New("only sandbox adapters are implemented; production startup refused")
	}
	if c.Profile != "" && len(c.ProfileToken) < 16 {
		return c, errors.New("PROFILE_INTERNAL_TOKEN required for the real Profile adapter")
	}
	if c.DB == "" || c.Redis == "" || c.Dependencies == "" || len(c.InternalToken) < 16 || len(c.JWTSecret) < 32 || len(c.CallbackSecret) < 32 || c.Receiver == "" {
		return c, errors.New("missing required configuration")
	}
	return c, nil
}

type Service struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	C        Config
	Client   *http.Client
	AEAD     cipher.AEAD
	Writer   *kafka.Writer
	workerID string
	register *prometheus.Registry
	Metrics  *prometheus.HistogramVec
	stop     sync.WaitGroup
}

func New(ctx context.Context, c Config) (*Service, error) {
	if c.Mode != "sandbox" {
		return nil, errors.New("production adapter set is not implemented")
	}
	key, e := base64.StdEncoding.DecodeString(c.EncryptionKey)
	if e != nil || len(key) != 32 {
		return nil, errors.New("ORDER_ENCRYPTION_KEY must be base64 32 bytes")
	}
	b, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	a, e := cipher.NewGCM(b)
	if e != nil {
		return nil, e
	}
	db, e := pgxpool.New(ctx, c.DB)
	if e != nil {
		return nil, e
	}
	if e = db.Ping(ctx); e != nil {
		db.Close()
		return nil, e
	}
	s := &Service{DB: db, Redis: redis.NewClient(&redis.Options{Addr: c.Redis, DB: c.RedisDB}), C: c, Client: &http.Client{Timeout: 2 * time.Second}, AEAD: a, workerID: uuid.NewString(), register: prometheus.NewRegistry()}
	s.Metrics = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "order_latency_seconds", Help: "Measured latency by phase/outcome", Buckets: []float64{.005, .01, .05, .1, .2, .3, .5, 1, 2, 5, 15, 60, 300, 900}}, []string{"phase", "outcome"})
	s.register.MustRegister(s.Metrics, &backlogCollector{S: s, Age: prometheus.NewDesc("order_backlog_oldest_seconds", "Oldest pending obligation age", []string{"phase"}, nil), Count: prometheus.NewDesc("order_backlog_count", "Pending obligations", []string{"phase"}, nil)})
	if c.Kafka != "" {
		s.Writer = &kafka.Writer{Addr: kafka.TCP(strings.Split(c.Kafka, ",")...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, WriteTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, AllowAutoTopicCreation: true}
	}
	return s, nil
}
func (s *Service) Close() {
	s.stop.Wait()
	if s.Writer != nil {
		_ = s.Writer.Close()
	}
	_ = s.Redis.Close()
	s.DB.Close()
}
func (s *Service) Seal(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, s.AEAD.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return nil, e
	}
	return s.AEAD.Seal(nonce, nonce, b, []byte("order-v3.1")), nil
}
func (s *Service) Open(b []byte, v any) error {
	n := s.AEAD.NonceSize()
	if len(b) < n {
		return errors.New("invalid ciphertext")
	}
	data, e := s.AEAD.Open(nil, b[:n], b[n:], []byte("order-v3.1"))
	if e != nil {
		return e
	}
	return json.Unmarshal(data, v)
}
func (s *Service) Call(ctx context.Context, method, path string, in, out any) error {
	return s.callURL(ctx, s.C.Dependencies, method, path, in, out)
}
func (s *Service) callURL(ctx context.Context, base, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
	}
	r, e := http.NewRequestWithContext(ctx, method, base+path, body)
	if e != nil {
		return e
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Internal-Token", s.C.InternalToken)
	resp, e := s.Client.Do(r)
	if e != nil {
		return domain.E("DEPENDENCY_UNKNOWN", 503)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var d struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&d)
		if d.Code == "" {
			d.Code = "DEPENDENCY_UNKNOWN"
		}
		return domain.E(d.Code, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
	}
	return nil
}
func (s *Service) ErrorCode(err error) (string, int) {
	var d *domain.Error
	if errors.As(err, &d) {
		return d.Code, d.HTTP
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "NOT_FOUND", 404
	}
	return "INTERNAL_ERROR", 500
}
func (s *Service) Resolve(ctx context.Context, p domain.Principal) (domain.Principal, error) {
	if p.Role == "GUEST" || p.Staff() {
		return p, nil
	}
	if p.Role != "CUSTOMER" {
		return p, domain.E("PERMISSION_DENIED", 403)
	}
	if s.C.Profile != "" {
		r, e := http.NewRequestWithContext(ctx, "GET", s.C.Profile+"/api/v1/profile", nil)
		if e != nil {
			return p, e
		}
		r.Header.Set("X-Internal-Token", s.C.ProfileToken)
		r.Header.Set("X-User-ID", p.ID)
		r.Header.Set("X-User-Role", "CUSTOMER")
		resp, e := s.Client.Do(r)
		if e != nil {
			return p, domain.E("IDENTITY_RESOLUTION_UNAVAILABLE", 503)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return p, domain.E("IDENTITY_RESOLUTION_UNAVAILABLE", 503)
		}
		var v struct {
			ID     string `json:"id"`
			UserID string `json:"user_id"`
		}
		if e = json.NewDecoder(resp.Body).Decode(&v); e != nil || v.UserID != p.ID {
			return p, domain.E("IDENTITY_RESOLUTION_UNAVAILABLE", 503)
		}
		p.CustomerID = v.ID
	} else {
		var v struct {
			CustomerID string `json:"customer_id"`
		}
		if e := s.Call(ctx, "GET", "/profile/"+p.ID, nil, &v); e != nil {
			return p, e
		}
		p.CustomerID = v.CustomerID
	}
	if _, e := uuid.Parse(p.CustomerID); e != nil {
		return p, domain.E("IDENTITY_RESOLUTION_UNAVAILABLE", 503)
	}
	return p, nil
}
func (s *Service) Audit(ctx context.Context, tx pgx.Tx, orderID, actor, action string, details any) error {
	if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(440031)"); e != nil {
		return e
	}
	var prev string
	e := tx.QueryRow(ctx, "SELECT hash FROM audit_records ORDER BY sequence DESC LIMIT 1").Scan(&prev)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	id := uuid.NewString()
	b, e := json.Marshal(details)
	if e != nil {
		return e
	}
	mac := hmac.New(sha256.New, []byte(s.C.CallbackSecret))
	fmt.Fprintf(mac, "%s|%s|%s|%s|%s|%s", prev, id, orderID, actor, action, b)
	_, e = tx.Exec(ctx, "INSERT INTO audit_records(id,order_id,actor,action,details,prev_hash,hash) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7)", id, orderID, actor, action, b, prev, hex.EncodeToString(mac.Sum(nil)))
	return e
}
func (s *Service) Emit(ctx context.Context, tx pgx.Tx, id, aggregate string, version int64, ordinal int, kind, status, payment, stock string, generation int64) error {
	eid := uuid.NewString()
	event := map[string]any{"specversion": "1.0", "id": eid, "source": "https://omama.vn/services/order-service", "type": "vn.omama.order." + kind + ".v2", "subject": strings.ToLower(aggregate) + ":" + id, "time": time.Now().UTC().Format(time.RFC3339Nano), "datacontenttype": "application/json", "data": map[string]any{"aggregate_id": id, "version": version, "status": status, "payment_status": payment, "stock_status": stock, "ready_generation": generation}}
	b, e := json.Marshal(event)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO outbox(id,aggregate_id,aggregate_type,version,ordinal,event_type,payload) VALUES($1,$2,$3,$4,$5,$6,$7)", eid, id, aggregate, version, ordinal, event["type"], b)
	return e
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
