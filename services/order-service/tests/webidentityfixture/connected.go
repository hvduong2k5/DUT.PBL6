package main

// Disposable acceptance environment. Catalog/Fulfillment and SePay statements
// are explicit TEST providers; Order, Inventory, Shipping and their DBs are real.
import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	shipping "github.com/omamx/order-service/internal/gen/shipping/v2"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"github.com/omamx/order-service/internal/simulator"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/grpc/metadata"
)

type fixtureTransport struct{ target *url.URL }

func (t fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = t.target.Scheme
	u.Host = t.target.Host
	copy.URL = &u
	return http.DefaultTransport.RoundTrip(copy)
}

func connected(ctx context.Context, cfg app.Config) (*app.Service, error) {
	cfg.DB = os.Getenv("WEB_TEST_ORDER_DATABASE")
	cfg.Redis = os.Getenv("WEB_TEST_REDIS")
	cfg.InternalToken = os.Getenv("WEB_TEST_INTERNAL_TOKEN")
	cfg.JWTSecret = strings.Repeat("x", 32)
	cfg.CallbackSecret = strings.Repeat("y", 32)
	cfg.EncryptionKey = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{17}, 32))
	cfg.Receiver = "SYNTHETIC-ONLY"
	cfg.Poll = 100 * time.Millisecond
	cfg.Kafka = os.Getenv("KAFKA_BROKERS")
	cfg.ConsumerGroup = "web-acceptance-" + uuid.NewString()
	sim, err := simulator.New(ctx, os.Getenv("WEB_TEST_CATALOG_DATABASE"), cfg.InternalToken, cfg.JWTSecret, cfg.CallbackSecret, cfg.Receiver)
	if err != nil {
		return nil, err
	}
	sim.OrderURL = "http://127.0.0.1:" + os.Getenv("WEB_TEST_ORDER_PORT")
	_, err = sim.DB.Exec(ctx, `INSERT INTO catalog(sku,name,price,weight,active,version) VALUES('PEER','Synthetic TEST SKU',1000,500,true,1) ON CONFLICT(sku) DO NOTHING`)
	if err != nil {
		return nil, err
	}
	original := httptest.NewServer(sim.Handler())
	// Only warehouse ID differs from the historical Catalog TEST provider.
	terms := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequestWithContext(r.Context(), r.Method, original.URL+r.URL.RequestURI(), r.Body)
		req.Header = r.Header.Clone()
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			http.Error(w, "test dependency unavailable", 503)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		if r.URL.Path == "/order-core/terms" && resp.StatusCode == 200 {
			var t core.SnapshotInput
			_ = json.NewDecoder(resp.Body).Decode(&t)
			t.WarehouseID = "00000000-0000-0000-0000-000000000001"
			_ = json.NewEncoder(w).Encode(t)
		} else {
			var v any
			_ = json.NewDecoder(resp.Body).Decode(&v)
			_ = json.NewEncoder(w).Encode(v)
		}
	}))
	cfg.Dependencies = terms.URL
	service, err := app.New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err = migrations.Apply(ctx, service.DB); err != nil {
		return nil, err
	}
	receiver := sepay.Receiver{BankID: uuid.NewString(), BankCode: "VCB", BankName: "Synthetic Bank", Account: "0000000000", Holder: "SYNTHETIC ONLY", Mode: "BANK_ACCOUNT", Currency: "VND"}
	folder := os.Getenv("WEB_TEST_FIXTURE_DIR")
	raw, _ := json.Marshal(receiver)
	for key, data := range map[string][]byte{"token": []byte(strings.Repeat("a", 64)), "secret": []byte(strings.Repeat("s", 32)), "receiver": raw} {
		path := filepath.Join(folder, "sepay-"+key)
		if err = os.WriteFile(path, data, 0600); err != nil {
			return nil, err
		}
		_ = os.Setenv("SEPAY_"+strings.ToUpper(key)+map[string]string{"token": "_FILE", "secret": "_FILE", "receiver": "_FILE"}[key], path)
	}
	_ = os.Setenv("SEPAY_API_TOKEN_FILE", filepath.Join(folder, "sepay-token"))
	_ = os.Setenv("SEPAY_WEBHOOK_SECRET_FILE", filepath.Join(folder, "sepay-secret"))
	for key, value := range map[string]string{"SEPAY_ENABLED": "true", "ORDER_CORE_ENABLED": "true", "ORDER_MERCHANT_ID": uuid.NewString(), "SEPAY_BOOTSTRAP_AT": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), "SEPAY_CONNECTION_ID": uuid.NewString(), "SEPAY_RECEIVER_ID": uuid.NewString(), "SEPAY_EGRESS_SCOPE": uuid.NewString(), "SEPAY_MERCHANT_IDENTITY": "synthetic-acceptance"} {
		_ = os.Setenv(key, value)
	}
	if err = service.EnableSePayFromEnv(ctx); err != nil {
		return nil, err
	}
	if err = service.EnableCoreCheckout(); err != nil {
		return nil, err
	}
	var mu sync.Mutex
	rows := []sepay.Transaction{}
	statements := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		last := 1
		if len(rows) == 0 {
			last = 0
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": rows, "meta": map[string]any{"pagination": map[string]any{"total": len(rows), "current_page": 1, "last_page": last, "per_page": 100, "has_more": false}}})
	}))
	target, _ := url.Parse(statements.URL)
	service.PaymentReader, err = sepay.NewSandboxClientWithTransport(strings.Repeat("a", 64), fixtureTransport{target})
	if err != nil {
		return nil, err
	}
	control := http.NewServeMux()
	control.HandleFunc("/prepare-cart", func(w http.ResponseWriter, r *http.Request) {
		p := domain.Principal{ID: os.Getenv("WEB_TEST_PRINCIPAL_A"), Role: "CUSTOMER"}
		cart, err := service.Cart(r.Context(), p)
		if err != nil {
			http.Error(w, "cart unavailable", 503)
			return
		}
		if len(cart.Items) != 0 {
			http.Error(w, "test cart must be empty", 409)
			return
		}
		if _, err = service.MutateCart(r.Context(), p, "PEER", 2, cart.Revision, true); err != nil {
			http.Error(w, "cart preparation failed", 503)
			return
		}
		w.WriteHeader(204)
	})
	control.HandleFunc("/pay", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("order")
		var scope string
		if err := service.DB.QueryRow(r.Context(), "SELECT scope FROM orders WHERE id=$1", id).Scan(&scope); err != nil {
			http.Error(w, "unknown test order", 404)
			return
		}
		instruction, err := service.Payment.Instructions(r.Context(), id, scope)
		if err != nil {
			http.Error(w, "instruction unavailable", 409)
			return
		}
		mu.Lock()
		rows = append(rows, sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: instruction.Amount, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: instruction.Code, Reference: uuid.NewString()})
		mu.Unlock()
		w.WriteHeader(202)
	})
	control.HandleFunc("/pack", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("order")
		p := domain.Principal{ID: os.Getenv("WEB_TEST_PRINCIPAL_A"), Role: "CUSTOMER", CustomerID: os.Getenv("WEB_TEST_CUSTOMER_A")}
		o, e := service.GetOrder(r.Context(), id, p, true)
		if e != nil || !o.Ready || o.CoreSnapshot == nil {
			http.Error(w, "test order is not ready", 409)
			return
		}
		task := uuid.NewString()
		seal := "synthetic-seal-" + id
		for i, kind := range []string{"PACKING_ACCEPTED", "PACKING_SEALED"} {
			source := persistence.SourceEvidence{Order: id, Resource: task, Source: "fulfillment", Kind: kind, Version: int64(i + 1), Generation: o.Generation}
			if i == 1 {
				source.Seal = seal
			}
			b, _ := json.Marshal(source)
			proofRequest, _ := http.NewRequestWithContext(r.Context(), "POST", original.URL+"/test/core-source", bytes.NewReader(b))
			proofRequest.Header.Set("Content-Type", "application/json")
			proofRequest.Header.Set("X-Internal-Token", cfg.InternalToken)
			resp, err := http.DefaultClient.Do(proofRequest)
			if err != nil {
				http.Error(w, "test packing proof unavailable", 503)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != 200 && resp.StatusCode != 201 {
				http.Error(w, fmt.Sprint("TEST packing provider rejected proof: ", resp.StatusCode), 503)
				return
			}
			event := domain.Event{ID: uuid.NewString(), Source: "fulfillment", OrderID: id, ResourceID: task, Type: kind, Version: int64(i + 1), Generation: o.Generation, Seal: source.Seal}
			event.Signature = domain.EventSignature(event, cfg.InternalToken)
			if state, err := service.ProcessEvent(r.Context(), event); err != nil || state != "APPLIED" {
				http.Error(w, fmt.Sprint("test packing ", state, err), 409)
				return
			}
		}
		t := o.CoreSnapshot
		destination := &common.Address{RecipientName: t.Address.RecipientName, PhoneNumber: t.Address.PhoneE164, StreetAddress: t.Address.Street, Ward: t.Address.WardName, Province: t.Address.ProvinceName, CountryCode: "VN"}
		rpcCtx := metadata.AppendToOutgoingContext(r.Context(), "authorization", "Bearer "+service.Shipping.Token)
		out, err := service.Shipping.Client.CreateShipment(rpcCtx, &shipping.CreateShipmentRequest{IdempotencyKey: "web-package-" + id, OrderId: id, PackageId: uuid.NewString(), SealCode: seal, ReadyGeneration: o.Generation, ItemsDigest: string(t.Provenance.ItemsDigest), OrderSnapshotVersion: 1, DestinationAddress: destination, ParcelWeight: &common.Weight{DecimalValue: "1000", Unit: common.WeightUnit_WEIGHT_UNIT_GRAM}, Carrier: "FAKE", ServiceCode: "STANDARD", CodAmount: &common.Money{CurrencyCode: "VND"}, CheckoutShippingFee: &common.Money{CurrencyCode: "VND", Units: t.Money.ShippingFee}})
		if err != nil {
			http.Error(w, err.Error(), 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"shipment_id": out.ShipmentId, "tracking_code": out.TrackingCode})
	})
	go http.ListenAndServe("127.0.0.1:"+os.Getenv("WEB_TEST_CONTROL_PORT"), control)
	service.Start(ctx)
	return service, nil
}
