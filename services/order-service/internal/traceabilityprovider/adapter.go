// Package traceabilityprovider resolves public links without participating in Order lifecycle commands.
package traceabilityprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	trace "github.com/omamx/order-service/internal/gen/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type Binding struct{ Batch, SKU string }
type Link struct {
	Status string
	URL    *string
}
type Adapter struct {
	Client              trace.BatchLinkResolverClient
	Token, PublicOrigin string
}

func Dial(target, token, ca, serverName, publicOrigin string) (*Adapter, *grpc.ClientConn, error) {
	if len(token) < 32 || target == "" || serverName == "" {
		return nil, nil, errors.New("Traceability requires a dedicated credential and TLS server name")
	}
	origin, err := url.Parse(publicOrigin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" && origin.Path != "/" {
		return nil, nil, errors.New("Traceability requires an HTTPS public origin")
	}
	certificate, err := os.ReadFile(ca)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certificate) {
		return nil, nil, errors.New("invalid Traceability CA")
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: serverName, MinVersion: tls.VersionTLS12})))
	if err != nil {
		return nil, nil, err
	}
	return &Adapter{Client: trace.NewBatchLinkResolverClient(conn), Token: token, PublicOrigin: strings.TrimSuffix(publicOrigin, "/")}, conn, nil
}

var labelPath = regexp.MustCompile(`^/trace/[A-Za-z0-9_-]{1,160}$`)

func (a *Adapter) validURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Scheme+"://"+u.Host == a.PublicOrigin && u.User == nil && u.RawQuery == "" && u.Fragment == "" && labelPath.MatchString(u.Path) && u.RawPath == ""
}

// Resolve keeps unresolved chunks explicit. At most two calls run concurrently,
// with a 1.5-second RPC budget inside the caller's overall read budget.
func (a *Adapter) Resolve(ctx context.Context, bindings []Binding) []Link {
	out := make([]Link, len(bindings))
	for i := range out {
		out[i].Status = "UNAVAILABLE"
	}
	if a == nil || a.Client == nil || len(bindings) > 1000 {
		return out
	}
	var next int
	var mu sync.Mutex
	var workers sync.WaitGroup
	for worker := 0; worker < 2; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				mu.Lock()
				start := next
				next += 100
				mu.Unlock()
				if start >= len(bindings) || ctx.Err() != nil {
					return
				}
				end := start + 100
				if end > len(bindings) {
					end = len(bindings)
				}
				items := make([]*trace.TraceabilityBatchBinding, end-start)
				for i, binding := range bindings[start:end] {
					items[i] = &trace.TraceabilityBatchBinding{SourceNamespace: "inventory", BatchId: binding.Batch, SkuCode: binding.SKU}
				}
				request := &trace.ResolveBatchLinksRequest{Items: items}
				if proto.Size(request) > 256<<10 {
					continue
				}
				call, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
				call = metadata.NewOutgoingContext(call, metadata.Pairs("authorization", "Bearer "+a.Token))
				response, err := a.Client.ResolveBatchLinks(call, request, grpc.MaxCallRecvMsgSize(512<<10), grpc.MaxCallSendMsgSize(256<<10))
				cancel()
				if err != nil || response == nil || len(response.Items) != len(items) || response.Total != int32(len(items)) {
					continue
				}
				links := make([]Link, len(items))
				valid := true
				for i, item := range response.Items {
					if item == nil || !proto.Equal(item.Binding, items[i]) {
						valid = false
						break
					}
					switch item.Status {
					case trace.BatchLinkStatus_BATCH_LINK_STATUS_AVAILABLE, trace.BatchLinkStatus_BATCH_LINK_STATUS_NO_LONGER_PUBLIC:
						if item.QrUrl == nil || !a.validURL(*item.QrUrl) {
							valid = false
							break
						}
						links[i] = Link{Status: strings.TrimPrefix(item.Status.String(), "BATCH_LINK_STATUS_"), URL: item.QrUrl}
					case trace.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE, trace.BatchLinkStatus_BATCH_LINK_STATUS_UNAVAILABLE:
						if item.QrUrl != nil {
							valid = false
							break
						}
						links[i].Status = strings.TrimPrefix(item.Status.String(), "BATCH_LINK_STATUS_")
					default:
						valid = false
					}
					if !valid {
						break
					}
				}
				if valid {
					copy(out[start:end], links)
				}
			}
		}()
	}
	workers.Wait()
	return out
}
