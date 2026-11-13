package adapters

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

func DialPeer(target, ca, serverName string, local bool) (*grpc.ClientConn, error) {
	var transport credentials.TransportCredentials
	if ca != "" {
		raw, e := os.ReadFile(ca)
		if e != nil {
			return nil, e
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(raw) || serverName == "" {
			return nil, errors.New("peer CA and server name required")
		}
		transport = credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: serverName, MinVersion: tls.VersionTLS12})
	} else if local {
		transport = insecure.NewCredentials()
	} else {
		return nil, errors.New("production peers require pinned TLS CA and server name")
	}
	return grpc.NewClient(target, grpc.WithTransportCredentials(transport))
}
