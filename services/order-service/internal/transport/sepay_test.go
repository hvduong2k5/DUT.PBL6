package transport

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"github.com/omamx/order-service/migrations"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func webhookServer(t *testing.T) (*httptest.Server, *pgxpool.Pool, *persistence.Payment) {
	t.Helper()
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("NOT_RUN: real PostgreSQL is required")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "payment_http_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	config, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, config)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		db.Close()
		_, e := admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		admin.Close()
		if e != nil {
			t.Error(e)
		}
	})
	if e = migrations.Apply(ctx, db); e != nil {
		t.Fatal(e)
	}
	cipher, e := persistence.NewCipher("test", map[string][]byte{"test": bytes.Repeat([]byte{1}, 32)})
	if e != nil {
		t.Fatal(e)
	}
	store, e := persistence.NewStore(db, cipher, bytes.Repeat([]byte{2}, 32))
	if e != nil {
		t.Fatal(e)
	}
	payment := &persistence.Payment{Store: store, Connection: uuid.NewString(), Receiver: uuid.NewString(), Egress: "http-test", Secret: strings.Repeat("s", 32), KeyVersion: "test-v1", Bootstrap: time.Now().Add(-time.Minute)}
	receiver := sepay.Receiver{BankID: uuid.NewString(), BankCode: "VCB", BankName: "Vietcombank", Account: "0000000000", Holder: "SYNTHETIC ONLY", Mode: "BANK_ACCOUNT", Currency: "VND"}
	raw, _ := json.Marshal(receiver)
	encrypted, e := cipher.Seal("receiver:"+payment.Receiver, raw)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, "INSERT INTO payment_provider_connections(id,provider,merchant_identity,environment,secret_ref) VALUES($1,'SEPAY',$2,'TEST','test')", payment.Connection, uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, "INSERT INTO payment_receiver_profiles(id,connection_id,bank_account_uuid,config_version,config_cipher,accepting_new_payments) VALUES($1,$2,$3,1,$4,true)", payment.Receiver, payment.Connection, receiver.BankID, encrypted); e != nil {
		t.Fatal(e)
	}
	if e = payment.Onboard(ctx, receiver); e != nil {
		t.Fatal(e)
	}
	router := chi.NewRouter()
	sepayRoutes(&app.Service{Payment: payment}, router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, db, payment
}
func TestSePayWebhookDurableHTTPIngress(t *testing.T) {
	server, db, payment := webhookServer(t)
	payload := []byte(`{"id":9007199254740993,"gateway":"VCB","transactionDate":"2026-10-24 10:00:00","accountNumber":"0000000000","content":"OMPABCDEFGHIJKL","transferType":"in","transferAmount":230000,"referenceCode":"SYNTHETIC-HTTP"}`)
	for _, kind := range []string{"unsigned", "media", "oversize", "valid", "replay", "invalid-json", "outgoing", "wrong-receiver", "rotation"} {
		t.Run(kind, func(t *testing.T) {
			raw := payload
			expected := 200
			media := "application/json"
			secret := payment.Secret
			switch kind {
			case "unsigned":
				secret = "wrong"
				expected = 401
			case "media":
				media = "text/plain"
				expected = 415
			case "oversize":
				raw = bytes.Repeat([]byte{'a'}, 65537)
				expected = 413
			case "invalid-json":
				raw = []byte(`{"id":1,"id":2}`)
				expected = 400
			case "outgoing":
				raw = bytes.ReplaceAll(payload, []byte(`9007199254740993`), []byte(`9007199254740994`))
				raw = bytes.ReplaceAll(raw, []byte(`"in"`), []byte(`"out"`))
			case "wrong-receiver":
				raw = bytes.ReplaceAll(payload, []byte(`9007199254740993`), []byte(`9007199254740995`))
				raw = bytes.ReplaceAll(raw, []byte(`0000000000`), []byte(`9999999999`))
			case "rotation":
				payment.PreviousSecret = strings.Repeat("r", 32)
				payment.PreviousKeyVersion = "previous-v1"
				payment.PreviousValidUntil = time.Now().Add(time.Minute)
				secret = payment.PreviousSecret
				raw = bytes.ReplaceAll(payload, []byte(`9007199254740993`), []byte(`9007199254740996`))
			}
			stamp := strconv.FormatInt(time.Now().Unix(), 10)
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(stamp + "."))
			mac.Write(raw)
			request, e := http.NewRequest("POST", server.URL+"/api/v1/payments/sepay/webhook", bytes.NewReader(raw))
			if e != nil {
				t.Fatal(e)
			}
			request.Header.Set("Content-Type", media)
			request.Header.Set("X-SePay-Timestamp", stamp)
			request.Header.Set("X-SePay-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
			response, e := http.DefaultClient.Do(request)
			if e != nil {
				t.Fatal(e)
			}
			response.Body.Close()
			if response.StatusCode != expected {
				t.Fatalf("status=%d expected=%d", response.StatusCode, expected)
			}
		})
	}
	var observations, receipts, review int
	if e := db.QueryRow(context.Background(), "SELECT (SELECT count(*) FROM payment_provider_observations),(SELECT count(*) FROM receipts),(SELECT count(*) FROM reconciliation_cases)").Scan(&observations, &receipts, &review); e != nil {
		t.Fatal(e)
	}
	if observations != 4 || receipts != 0 || review != 2 {
		t.Fatal("durable ingress invariant", observations, receipts, review)
	}
}
