package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/dependency"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	httptransport "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/transport/http"
	catalogv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/catalog/v1"
	identityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/identity/v1"
	mediav1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/media/v1"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"strconv"
)

func integrationAdapters(ctx context.Context, getenv func(string) string) (httptransport.IdentityVerifier, application.DependencyVerifier, func(), error) {
	catalogToken, inventoryToken, identityToken := getenv("TRACEABILITY_CATALOG_TOKEN"), getenv("TRACEABILITY_INVENTORY_TOKEN"), getenv("TRACEABILITY_IDENTITY_TOKEN")
	issuer := getenv("TRACEABILITY_OIDC_ISSUER")
	if len(catalogToken) < 32 || len(inventoryToken) < 32 || len(identityToken) < 32 || catalogToken == inventoryToken || catalogToken == identityToken || identityToken == inventoryToken || !strings.HasPrefix(issuer, "https://") {
		return nil, nil, nil, errors.New("integration requires distinct service credentials and HTTPS OIDC issuer")
	}
	// OIDC protocol traffic is HTTPS; business dependencies below are gRPC only.
	client, err := oidcHTTPClient(getenv("TRACEABILITY_OIDC_CA_FILE"))
	if err != nil {
		return nil, nil, nil, err
	}
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
		target, tls, err := peerTransport(getenv, prefix)
		if err != nil {
			return nil, err
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
	source := dependency.Live{CatalogSource: catalogv1.NewCatalogServiceClient(catalog), Inventory: traceabilityv1.NewBatchSourceClient(inventory), CatalogToken: catalogToken, InventoryToken: inventoryToken}
	if mediaEnabled, _ := strconv.ParseBool(getenv("TRACEABILITY_MEDIA_ENABLED")); mediaEnabled {
		media, err := dial("TRACEABILITY_MEDIA")
		if err != nil {
			return nil, nil, nil, err
		}
		source.MediaClient = mediav1.NewMediaInternalServiceClient(media)
	}
	assembled = true
	return verifier, source, cleanup, nil
}

// peerTransport follows the canonical provider config. Plaintext is a deliberate
// private-network compatibility mode, never an automatic reaction to TLS failure.
func peerTransport(getenv func(string) string, prefix string) (string, credentials.TransportCredentials, error) {
	target, ca := getenv(prefix+"_ADDR"), getenv(prefix+"_CA_FILE")
	if override := getenv(prefix + "_GRPC_ADDR"); override != "" {
		target = override
	}
	raw := getenv(prefix + "_GRPC_PLAINTEXT")
	plaintext, err := strconv.ParseBool(raw)
	if err != nil && raw != "" {
		return "", nil, errors.New("invalid peer plaintext flag")
	}
	if plaintext {
		host, port, splitErr := net.SplitHostPort(target)
		number, portErr := strconv.Atoi(port)
		ip := net.ParseIP(host)
		private := host == "localhost" || (prefix == "TRACEABILITY_CATALOG" && host == "catalog-service") || (prefix == "TRACEABILITY_MEDIA" && host == "media-service") || (ip != nil && (ip.IsLoopback() || ip.IsPrivate()))
		if (prefix != "TRACEABILITY_CATALOG" && prefix != "TRACEABILITY_MEDIA") || splitErr != nil || portErr != nil || number < 1 || number > 65535 || !private {
			return "", nil, errors.New("plaintext peer must use an explicit private Catalog/Media target")
		}
		return target, insecure.NewCredentials(), nil
	}
	if target == "" || ca == "" {
		return "", nil, errors.New("explicit gRPC target and TLS CA required")
	}
	tls, err := credentials.NewClientTLSFromFile(ca, "")
	if err != nil {
		return "", nil, errors.New("invalid peer TLS CA")
	}
	return target, tls, nil
}
