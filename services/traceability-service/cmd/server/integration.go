package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/dependency"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	httptransport "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/transport/http"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func integrationAdapters(ctx context.Context, getenv func(string) string) (httptransport.IdentityVerifier, application.DependencyVerifier, func(), error) {
	catalogURL, catalogErr := dependency.ServiceOrigin(getenv("TRACEABILITY_CATALOG_URL"))
	identityURL, identityErr := dependency.ServiceOrigin(getenv("TRACEABILITY_IDENTITY_URL"))
	catalogToken := getenv("TRACEABILITY_CATALOG_TOKEN")
	inventoryToken := getenv("TRACEABILITY_INVENTORY_TOKEN")
	issuer := getenv("TRACEABILITY_OIDC_ISSUER")
	target := getenv("TRACEABILITY_INVENTORY_ADDR")
	if catalogErr != nil || identityErr != nil || len(catalogToken) < 32 || len(inventoryToken) < 32 ||
		catalogToken == inventoryToken || !strings.HasPrefix(issuer, "https://") || target == "" {
		return nil, nil, nil, errors.New("integration requires explicit Catalog, Inventory and Identity configuration")
	}
	client := &http.Client{
		Timeout:       3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	discoveryCtx, cancel := context.WithTimeout(oidc.ClientContext(ctx, client), 5*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(discoveryCtx, issuer)
	if err != nil {
		return nil, nil, nil, errors.New("cannot discover configured OIDC issuer")
	}
	tlsCredentials, err := credentials.NewClientTLSFromFile(getenv("TRACEABILITY_INVENTORY_CA_FILE"), "")
	if err != nil {
		return nil, nil, nil, errors.New("invalid Inventory TLS CA")
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(tlsCredentials))
	if err != nil {
		return nil, nil, nil, errors.New("invalid Inventory gRPC configuration")
	}
	verifier := identity.Staff{Tokens: provider.Verifier(&oidc.Config{ClientID: "omama-api", SupportedSigningAlgs: []string{oidc.RS256}}), IdentityURL: identityURL, HTTP: client}
	sources := dependency.Live{CatalogURL: catalogURL, CatalogToken: catalogToken, InventoryToken: inventoryToken, HTTP: client, Inventory: traceabilityv1.NewBatchSourceClient(conn)}
	return verifier, sources, func() { _ = conn.Close() }, nil
}
