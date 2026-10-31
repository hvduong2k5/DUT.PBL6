package paymentinventory

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"time"
)

// DialTerminal never disables certificate/hostname validation or falls back to plaintext.
func DialTerminal(ctx context.Context, target, token, caFile, serverName string) (Adapter, *grpc.ClientConn, error) {
	if target == "" || len(token) < 32 || caFile == "" || serverName == "" {
		return Adapter{}, nil, fmt.Errorf("terminal Inventory requires target, dedicated token >=32 bytes, CA and server name")
	}
	raw, err := os.ReadFile(caFile)
	if err != nil {
		return Adapter{}, nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(raw) {
		return Adapter{}, nil, fmt.Errorf("invalid Inventory CA")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS12})), grpc.WithBlock())
	if err != nil {
		return Adapter{}, nil, err
	}
	return Adapter{Client: inventory.NewInventoryServiceClient(conn), Token: token, Terminal: true}, conn, nil
}
