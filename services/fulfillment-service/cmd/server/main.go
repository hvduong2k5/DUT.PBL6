package main

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/fulfillment-service/internal/adapters"
	"github.com/omamx/fulfillment-service/internal/application"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	p "github.com/omamx/fulfillment-service/internal/gen/packing"
	"github.com/omamx/fulfillment-service/internal/persistence"
	"github.com/omamx/fulfillment-service/internal/transport"
	"github.com/omamx/fulfillment-service/internal/worker"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func env(name, def string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return def
}
func enabled(name string) bool {
	v := env(name, "false")
	if v != "true" && v != "false" {
		log.Fatalf("%s must be true or false", name)
	}
	return v == "true"
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn := os.Getenv("FULFILLMENT_DATABASE_URL")
	if dsn == "" {
		log.Fatal("FULFILLMENT_DATABASE_URL required")
	}
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		log.Fatal(e)
	}
	defer pool.Close()
	if e = pool.Ping(ctx); e != nil {
		log.Fatal(e)
	}
	var auth transport.Auth
	auth.Issuer = os.Getenv("FULFILLMENT_JWT_ISSUER")
	auth.StaffAudience = env("FULFILLMENT_STAFF_AUDIENCE", "fulfillment-api")
	if path := os.Getenv("FULFILLMENT_ORDER_CALLER_TOKEN_FILE"); path != "" {
		raw, e := os.ReadFile(path)
		if e != nil || len(strings.TrimSpace(string(raw))) < 32 {
			log.Fatal("Order caller credential required")
		}
		auth.InternalTokens = map[string]string{"order-service": strings.TrimSpace(string(raw))}
	}
	if path := os.Getenv("FULFILLMENT_JWT_PUBLIC_KEY"); path != "" {
		raw, e := os.ReadFile(path)
		if e != nil {
			log.Fatal(e)
		}
		auth.Key, e = jwt.ParseRSAPublicKeyFromPEM(raw)
		if e != nil {
			log.Fatal(e)
		}
	}
	local := env("FULFILLMENT_ENV", "production") == "local"
	certificate, key := os.Getenv("FULFILLMENT_TLS_CERT"), os.Getenv("FULFILLMENT_TLS_KEY")
	if !local && (certificate == "" || key == "" || auth.Key == nil || auth.Issuer == "") {
		log.Fatal("production requires TLS certificate/key and JWT issuer/public key")
	}
	store := &persistence.Store{DB: pool}
	svc := &application.Service{Store: store, Authority: application.Unavailable{}, AdmissionEnabled: enabled("FULFILLMENT_ADMISSION_ENABLED"), SealEnabled: enabled("FULFILLMENT_SEAL_ENABLED"), Integrated: enabled("FULFILLMENT_PACKING_V2_ENABLED")}
	if svc.Integrated && (auth.Key == nil || auth.Issuer == "") {
		log.Fatal("integrated staff APIs require verified JWT issuer and public key")
	}
	if enabled("FULFILLMENT_REOPEN_ENABLED") || enabled("FULFILLMENT_MULTI_PACKAGE_ENABLED") || enabled("FULFILLMENT_CUSTOMER_MEDIA_ENABLED") {
		log.Fatal("unsupported capability flag")
	}
	if enabled("FULFILLMENT_CANDIDATE_PEERS_ENABLED") {
		oe, ie, token := os.Getenv("FULFILLMENT_ORDER_GRPC_ENDPOINT"), os.Getenv("FULFILLMENT_INVENTORY_GRPC_ENDPOINT"), os.Getenv("FULFILLMENT_PEER_BEARER_TOKEN")
		if path := os.Getenv("FULFILLMENT_ORDER_TOKEN_FILE"); path != "" {
			raw, e := os.ReadFile(path)
			if e != nil {
				log.Fatal(e)
			}
			token = strings.TrimSpace(string(raw))
		}
		if oe == "" || ie == "" || token == "" {
			log.Fatal("candidate peer endpoints and token required")
		}
		oc, e := adapters.DialPeer(oe, os.Getenv("FULFILLMENT_ORDER_CA_FILE"), os.Getenv("FULFILLMENT_ORDER_SERVER_NAME"), local)
		if e != nil {
			log.Fatal(e)
		}
		defer oc.Close()
		ic, e := adapters.DialPeer(ie, os.Getenv("FULFILLMENT_INVENTORY_CA_FILE"), os.Getenv("FULFILLMENT_INVENTORY_SERVER_NAME"), local)
		if e != nil {
			log.Fatal(e)
		}
		defer ic.Close()
		peers := adapters.NewPeers(oc, ic, token)
		if path := os.Getenv("FULFILLMENT_INVENTORY_TOKEN_FILE"); path != "" {
			raw, e := os.ReadFile(path)
			if e != nil || len(strings.TrimSpace(string(raw))) < 32 {
				log.Fatal("Inventory packing credential required")
			}
			peers.InventoryToken = strings.TrimSpace(string(raw))
		}
		if svc.Integrated && (len(token) < 32 || peers.InventoryToken == "" || len(auth.InternalTokens) == 0) {
			log.Fatal("integrated profile requires separate Order/Inventory peer credentials")
		}
		svc.Authority = peers
	} else if svc.AdmissionEnabled || svc.SealEnabled {
		log.Fatal("mutation capabilities require configured peer adapters")
	}
	domainTest := enabled("FULFILLMENT_DOMAIN_TEST_MODE")
	if domainTest && !local {
		log.Fatal("domain-test profile requires isolated local environment")
	}
	if (svc.AdmissionEnabled || svc.SealEnabled) && !svc.Integrated && !domainTest {
		log.Fatal("real mutation profile requires integrated-v2; domain-only mode is isolated test only")
	}
	media := &application.Media{Store: store, UploadEnabled: enabled("FULFILLMENT_MEDIA_UPLOAD_ENABLED"), AccessEnabled: enabled("FULFILLMENT_MEDIA_ACCESS_ENABLED")}
	if media.UploadEnabled || media.AccessEnabled || os.Getenv("FULFILLMENT_STORAGE_ENDPOINT") != "" {
		bucket, endpoint := os.Getenv("FULFILLMENT_STORAGE_BUCKET"), os.Getenv("FULFILLMENT_STORAGE_ENDPOINT")
		if bucket == "" || endpoint == "" {
			log.Fatal("private media bucket and endpoint required")
		}
		storage, e := adapters.NewS3(endpoint, os.Getenv("FULFILLMENT_STORAGE_ACCESS_KEY"), os.Getenv("FULFILLMENT_STORAGE_SECRET_KEY"), bucket, !local)
		if e != nil {
			log.Fatal(e)
		}
		media.Storage = storage
		if media.UploadEnabled || os.Getenv("FULFILLMENT_CLAMD_ADDR") != "" {
			address := os.Getenv("FULFILLMENT_CLAMD_ADDR")
			if address == "" {
				log.Fatal("real scanner required")
			}
			if _, e = exec.LookPath("ffprobe"); e != nil {
				log.Fatal("ffprobe required")
			}
			media.Scanner = adapters.Clamd{Address: address}
		}
	}
	if (svc.AdmissionEnabled || svc.SealEnabled || media.UploadEnabled || media.AccessEnabled) && (auth.Key == nil || auth.Issuer == "") {
		log.Fatal("mutation/media capabilities require verified JWT identity configuration")
	}
	if svc.Integrated && (!svc.AdmissionEnabled || !svc.SealEnabled || !media.UploadEnabled) {
		log.Fatal("integrated-v2 requires admission/seal/verified-media capabilities")
	}
	runtime := &worker.Runtime{Store: store, Media: media, Service: svc}
	if brokers := os.Getenv("FULFILLMENT_KAFKA_BROKERS"); brokers != "" {
		runtime.Kafka = &kafka.Writer{Addr: kafka.TCP(strings.Split(brokers, ",")...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, Async: false, WriteTimeout: 5 * time.Second}
		defer runtime.Kafka.Close()
		if svc.AdmissionEnabled {
			go runtime.ConsumeReady(ctx, kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(brokers, ","), Topic: "order.events.v3", GroupID: "fulfillment-ready-v1", StartOffset: kafka.FirstOffset, CommitInterval: 0, MaxBytes: 1048576}))
		}
	}
	go runtime.Run(ctx)
	rpc := &transport.RPC{Service: svc, Auth: auth}
	options := []grpc.ServerOption{grpc.UnaryInterceptor(auth.Interceptor), grpc.MaxRecvMsgSize(65536)}
	if !local {
		creds, e := credentials.NewServerTLSFromFile(certificate, key)
		if e != nil {
			log.Fatal(e)
		}
		options = append(options, grpc.Creds(creds))
	}
	grpcServer := grpc.NewServer(options...)
	f.RegisterFulfillmentServiceServer(grpcServer, rpc)
	p.RegisterFulfillmentPackageProofCandidateServer(grpcServer, rpc)
	listener, e := net.Listen("tcp", env("FULFILLMENT_GRPC_ADDR", ":9002"))
	if e != nil {
		log.Fatal(e)
	}
	go func() {
		if e := grpcServer.Serve(listener); e != nil {
			log.Printf("gRPC stopped: %v", e)
			stop()
		}
	}()
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "fulfillment_outbox_oldest_seconds", Help: "Age of oldest unacknowledged outbox event"}, func() float64 {
		c, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		v, e := store.OutboxAge(c)
		if e != nil {
			return -1
		}
		return v
	}))
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.Handle("/", (transport.HTTP{Service: svc, Media: media, Auth: auth}).Router())
	httpServer := &http.Server{Addr: env("FULFILLMENT_HTTP_ADDR", ":8002"), Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		var e error
		if local {
			e = httpServer.ListenAndServe()
		} else {
			e = httpServer.ListenAndServeTLS(certificate, key)
		}
		if e != nil && e != http.ErrServerClosed {
			log.Printf("HTTP stopped: %v", e)
			stop()
		}
	}()
	log.Print("Fulfillment started; capabilities remain explicitly gated")
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdown)
	done := make(chan struct{})
	go func() { grpcServer.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-shutdown.Done():
		grpcServer.Stop()
	}
}
