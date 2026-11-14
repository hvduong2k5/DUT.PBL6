package promotionprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"time"

	api "github.com/omamx/order-service/internal/gen/promotion/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(ctx context.Context, target, token, ca, server string, local bool) (*Adapter, *grpc.ClientConn, error) {
	if len(token) < 32 {
		return nil, nil, errors.New("Promotion requires a service token of at least 32 bytes")
	}
	var transport credentials.TransportCredentials
	if local {
		host, _, err := net.SplitHostPort(target)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) || ca != "" || server != "" {
			return nil, nil, errors.New("plaintext Promotion is allowed only on an explicit loopback test endpoint")
		}
		transport = insecure.NewCredentials()
	} else {
		if ca == "" || server == "" {
			return nil, nil, errors.New("Promotion requires a CA file and server name")
		}
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, nil, err
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return nil, nil, errors.New("invalid Promotion CA")
		}
		transport = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: server})
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(transport), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}
	return &Adapter{Client: api.NewPromotionServiceClient(conn), Token: token}, conn, nil
}
