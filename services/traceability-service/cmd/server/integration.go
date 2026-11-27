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
	catalogv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/catalog/v1"
	identityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/identity/v1"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func integrationAdapters(ctx context.Context, getenv func(string) string) (httptransport.IdentityVerifier, application.DependencyVerifier, func(), error) {
	catalogToken, inventoryToken, identityToken := getenv("TRACEABILITY_CATALOG_TOKEN"), getenv("TRACEABILITY_INVENTORY_TOKEN"), getenv("TRACEABILITY_IDENTITY_TOKEN")
	issuer := getenv("TRACEABILITY_OIDC_ISSUER")
	if len(catalogToken) < 32 || len(inventoryToken) < 32 || len(identityToken) < 32 || catalogToken == inventoryToken || catalogToken == identityToken || identityToken == inventoryToken || !strings.HasPrefix(issuer, "https://") {
		return nil, nil, nil, errors.New("integration requires distinct service credentials and HTTPS OIDC issuer")
	}
	// OIDC protocol traffic is HTTPS; business dependencies below are gRPC only.
	client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	discovery, cancel := context.WithTimeout(oidc.ClientContext(ctx, client), 5*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(discovery, issuer)
	if err != nil {
		return nil, nil, nil, errors.New("cannot discover configured OIDC issuer")
	}
	connections := []*grpc.ClientConn{}
	cleanup := func() {
		for _, conn := range connections {
			_ = conn.Close()
		}
	}
	assembled := false
	defer func() {
		if !assembled {
			cleanup()
		}
	}()
	dial := func(prefix string) (*grpc.ClientConn, error) {
		target, ca := getenv(prefix+"_ADDR"), getenv(prefix+"_CA_FILE")
		if target == "" || ca == "" {
			return nil, errors.New("explicit gRPC target and TLS CA required")
		}
		tls, err := credentials.NewClientTLSFromFile(ca, "")
		if err != nil {
			return nil, errors.New("invalid peer TLS CA")
		}
		conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(tls))
		if err != nil {
			return nil, errors.New("invalid peer gRPC configuration")
		}
		connections = append(connections, conn)
		return conn, nil
	}
	catalog, err := dial("TRACEABILITY_CATALOG")
	if err != nil {
		return nil, nil, nil, err
	}
	inventory, err := dial("TRACEABILITY_INVENTORY")
	if err != nil {
		return nil, nil, nil, err
	}
	iam, err := dial("TRACEABILITY_IDENTITY")
	if err != nil {
		return nil, nil, nil, err
	}
	verifier := identity.Staff{Tokens: provider.Verifier(&oidc.Config{ClientID: "omama-api", SupportedSigningAlgs: []string{oidc.RS256}}), Client: identityv1.NewStaffAuthorizationClient(iam), ServiceToken: identityToken, HTTP: client}
	source := dependency.Live{CatalogSource: catalogv1.NewCatalogTraceabilitySourceClient(catalog), Inventory: traceabilityv1.NewBatchSourceClient(inventory), CatalogToken: catalogToken, InventoryToken: inventoryToken}
	assembled = true
	return verifier, source, cleanup, nil
}
