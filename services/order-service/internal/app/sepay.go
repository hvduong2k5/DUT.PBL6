package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"github.com/omamx/order-service/internal/paymentauthority"
	"github.com/omamx/order-service/internal/paymentinventory"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"strings"
	"time"
)

// EnableSePayFromEnv runs after migrations and explicitly opts into TEST only.
// Stable connection/profile IDs are configuration, never regenerated on restart.
func (s *Service) EnableSePayFromEnv(ctx context.Context) error {
	if os.Getenv("SEPAY_ENABLED") != "true" {
		return nil
	}
	if s.C.Mode != "sandbox" {
		return errors.New("SePay runtime currently accepts TEST configuration only")
	}
	read := func(name string) ([]byte, error) {
		path := os.Getenv(name)
		if path == "" {
			return nil, errors.New("missing SePay secret/profile file")
		}
		return os.ReadFile(path)
	}
	token, e := read("SEPAY_API_TOKEN_FILE")
	if e != nil {
		return e
	}
	secret, e := read("SEPAY_WEBHOOK_SECRET_FILE")
	if e != nil {
		return e
	}
	receiverBytes, e := read("SEPAY_RECEIVER_FILE")
	if e != nil {
		return e
	}
	var receiver sepay.Receiver
	if sepay.Decode(receiverBytes, &receiver) != nil || receiver.Validate() != nil {
		return sepay.ErrInvalid
	}
	key, e := base64.StdEncoding.DecodeString(s.C.EncryptionKey)
	if e != nil {
		return e
	}
	cipher, e := persistence.NewCipher("core-v1", map[string][]byte{"core-v1": key})
	if e != nil {
		return e
	}
	store, e := persistence.NewStore(s.DB, cipher, []byte(s.C.CallbackSecret))
	if e != nil {
		return e
	}
	bootstrap, e := time.Parse(time.RFC3339, os.Getenv("SEPAY_BOOTSTRAP_AT"))
	if e != nil {
		return e
	}
	p := &persistence.Payment{Store: store, Connection: os.Getenv("SEPAY_CONNECTION_ID"), Receiver: os.Getenv("SEPAY_RECEIVER_ID"), Egress: os.Getenv("SEPAY_EGRESS_SCOPE"), Secret: string(secret), KeyVersion: "hmac-v1", Bootstrap: bootstrap}
	if os.Getenv("SEPAY_WEBHOOK_PREVIOUS_SECRET_FILE") != "" {
		previous, e := read("SEPAY_WEBHOOK_PREVIOUS_SECRET_FILE")
		if e != nil {
			return e
		}
		until, e := time.Parse(time.RFC3339, os.Getenv("SEPAY_WEBHOOK_PREVIOUS_VALID_UNTIL"))
		if e != nil || !until.After(time.Now()) || until.After(time.Now().Add(24*time.Hour)) {
			return persistence.ErrInput
		}
		p.PreviousSecret = string(previous)
		p.PreviousKeyVersion = os.Getenv("SEPAY_WEBHOOK_PREVIOUS_KEY_VERSION")
		p.PreviousValidUntil = until
		if len(previous) < 32 || p.PreviousKeyVersion == "" || p.PreviousKeyVersion == p.KeyVersion {
			return persistence.ErrInput
		}
	}
	if p.Validate() != nil || os.Getenv("SEPAY_MERCHANT_IDENTITY") == "" {
		return persistence.ErrInput
	}
	if os.Getenv("SEPAY_GUEST_SIGNING_KEY_FILE") != "" {
		raw, e := read("SEPAY_GUEST_SIGNING_KEY_FILE")
		if e != nil {
			return e
		}
		block, _ := pem.Decode(raw)
		if block == nil {
			return persistence.ErrInput
		}
		key, e := x509.ParsePKCS8PrivateKey(block.Bytes)
		if e != nil {
			return e
		}
		private, ok := key.(*ecdsa.PrivateKey)
		if !ok || private.Curve != elliptic.P256() || os.Getenv("SEPAY_GUEST_SIGNING_KEY_ID") == "" {
			return persistence.ErrInput
		}
		p.GrantKey = private
		p.GrantKeyID = os.Getenv("SEPAY_GUEST_SIGNING_KEY_ID")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `INSERT INTO payment_provider_connections(id,provider,merchant_identity,environment,secret_ref) VALUES($1,'SEPAY',$2,'TEST','SEPAY_WEBHOOK_SECRET_FILE') ON CONFLICT DO NOTHING`, p.Connection, os.Getenv("SEPAY_MERCHANT_IDENTITY")); e != nil {
		return e
	}
	var configuredMerchant string
	if e = tx.QueryRow(ctx, "SELECT merchant_identity FROM payment_provider_connections WHERE id=$1 AND provider='SEPAY' AND environment='TEST'", p.Connection).Scan(&configuredMerchant); e != nil {
		return e
	}
	if configuredMerchant != os.Getenv("SEPAY_MERCHANT_IDENTITY") {
		return persistence.ErrIntegrity
	}
	config, _ := json.Marshal(receiver)
	encrypted, e := cipher.Seal("receiver:"+p.Receiver, config)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO payment_receiver_profiles(id,connection_id,bank_account_uuid,config_version,config_cipher,accepting_new_payments) VALUES($1,$2,$3,1,$4,true) ON CONFLICT DO NOTHING`, p.Receiver, p.Connection, receiver.BankID, encrypted); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	if e = p.Onboard(ctx, receiver); e != nil {
		return e
	}
	client, e := sepay.NewSandboxClient(strings.TrimSpace(string(token)))
	if e != nil {
		return e
	}
	s.Payment = p
	s.PaymentReader = client
	if base := os.Getenv("SEPAY_CARE_URL"); base != "" {
		authority, e := paymentauthority.New(base, s.C.InternalToken)
		if e != nil {
			return e
		}
		s.PaymentApproval = authority
		s.PaymentCases = authority
	}
	if base := os.Getenv("SEPAY_FINANCE_URL"); base != "" {
		authority, e := paymentauthority.New(base, s.C.InternalToken)
		if e != nil {
			return e
		}
		s.PaymentEvidence = authority
		s.PaymentVault = authority
		for _, host := range strings.Split(os.Getenv("SEPAY_EVIDENCE_HOSTS"), ",") {
			if strings.TrimSpace(host) != "" {
				s.PaymentEvidenceHosts = append(s.PaymentEvidenceHosts, strings.TrimSpace(host))
			}
		}
	}
	if target := os.Getenv("SEPAY_INVENTORY_GRPC"); target != "" {
		conn, e := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if e != nil {
			return e
		}
		s.PaymentInventory = paymentinventory.Adapter{Client: inventory.NewInventoryServiceClient(conn), Token: s.C.InternalToken}
		s.PaymentClose = conn.Close
	}
	return nil
}
