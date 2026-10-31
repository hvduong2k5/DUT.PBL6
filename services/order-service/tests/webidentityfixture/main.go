// Test fixture: actual Order HTTP handler/authentication and real Redis. Seeded
// cart rows replace Catalog data; this fixture does not prove a checkout/payment.
package main

import (
	"context"
	"encoding/json"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/transport"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx := context.Background()
	cfg := app.Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: os.Getenv("WEB_TEST_ISSUER"), OIDCAudience: "order-api", OIDCAllowLocalHTTP: true, Profile: os.Getenv("WEB_TEST_PROFILE_URL"), ProfileToken: os.Getenv("WEB_TEST_INTERNAL_TOKEN")}
	identity, err := app.NewCustomerIdentity(cfg)
	if err != nil {
		log.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("WEB_TEST_REDIS")})
	defer rdb.Close()
	for _, suffix := range []string{"A", "B"} {
		p := domain.Principal{ID: os.Getenv("WEB_TEST_PRINCIPAL_" + suffix), Role: "CUSTOMER"}
		cart := domain.Cart{Revision: 1, Items: []domain.Item{{ItemID: "fixture-" + suffix, SKU: "PRIVATE-" + suffix, Quantity: 1, UnitPrice: 1000, Name: "Test cart " + suffix}}}
		raw, _ := json.Marshal(cart)
		if err = rdb.Set(ctx, "cart:"+p.Scope(), raw, time.Hour).Err(); err != nil {
			log.Fatal(err)
		}
	}
	service := &app.Service{C: cfg, Identity: identity, Redis: rdb}
	log.Fatal(http.ListenAndServe(":"+os.Getenv("WEB_TEST_ORDER_PORT"), transport.Handler(service)))
}
