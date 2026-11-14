package grpcapi

import (
	"strings"
	"testing"

	"github.com/omamx/promotion-service/internal/domain"
	commonv1 "github.com/omamx/promotion-service/internal/gen/common/v1"
	promotionv1 "github.com/omamx/promotion-service/internal/gen/promotion/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidateCommand(t *testing.T) {
	validHash := strings.Repeat("a", 64)
	tests := []struct {
		name, key, hash string
		wantErr         bool
	}{
		{name: "valid command", key: "order-command-key-0001", hash: validHash},
		{name: "key too short", key: "short", hash: validHash, wantErr: true},
		{name: "key contains whitespace", key: "invalid key command", hash: validHash, wantErr: true},
		{name: "hash wrong length", key: "order-command-key-0001", hash: "abcd", wantErr: true},
		{name: "hash must be hexadecimal", key: "order-command-key-0001", hash: strings.Repeat("z", 64), wantErr: true},
		{name: "hash must be lowercase", key: "order-command-key-0001", hash: strings.Repeat("A", 64), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.key, tt.hash)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateOrderVoucherCommands(t *testing.T) {
	valid := &promotionv1.VoucherHoldCommand{ContractVersion: orderContractVersion, OrderId: "order-1", QueryAction: "FinalizeVoucherHold", IdempotencyKey: "order-command-key-0001", RequestHash: strings.Repeat("b", 64)}
	tests := []struct {
		name    string
		request *promotionv1.VoucherHoldCommand
		query   bool
		wantErr bool
	}{
		{name: "valid command", request: valid},
		{name: "wrong contract", request: &promotionv1.VoucherHoldCommand{ContractVersion: "order-3.1", OrderId: "order-1", QueryAction: "ReleaseVoucherHoldByOrder", IdempotencyKey: valid.IdempotencyKey, RequestHash: valid.RequestHash}, wantErr: true},
		{name: "missing order", request: &promotionv1.VoucherHoldCommand{ContractVersion: orderContractVersion, QueryAction: "FinalizeVoucherHold", IdempotencyKey: valid.IdempotencyKey, RequestHash: valid.RequestHash}, wantErr: true},
		{name: "read-only query", request: &promotionv1.VoucherHoldCommand{ContractVersion: orderContractVersion, OrderId: "order-1"}, query: true},
		{name: "query must pair proof fields", request: &promotionv1.VoucherHoldCommand{ContractVersion: orderContractVersion, OrderId: "order-1", IdempotencyKey: valid.IdempotencyKey}, query: true, wantErr: true},
		{name: "query action must be supported", request: &promotionv1.VoucherHoldCommand{ContractVersion: orderContractVersion, OrderId: "order-1", QueryAction: "Unknown", IdempotencyKey: valid.IdempotencyKey, RequestHash: valid.RequestHash}, query: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.query {
				err = validateOrderQuery(tt.request)
			} else {
				err = validateOrderCommand(tt.request)
			}
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMoneyUnitsRequiresNonNegativeWholeVND(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   *commonv1.Money
		wantErr bool
	}{
		{name: "nil", wantErr: true},
		{name: "VND whole units", value: &commonv1.Money{CurrencyCode: "VND", Units: 42}},
		{name: "wrong currency", value: &commonv1.Money{CurrencyCode: "USD", Units: 42}, wantErr: true},
		{name: "fractional units", value: &commonv1.Money{CurrencyCode: "VND", Units: 42, Nanos: 1}, wantErr: true},
		{name: "negative units", value: &commonv1.Money{CurrencyCode: "VND", Units: -1}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			units, currency, err := moneyUnits(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || units != 42 || currency != "VND" {
				t.Fatalf("got %d %q %v", units, currency, err)
			}
		})
	}
}

func TestVoucherDomainErrorsMapToStableGRPCStatus(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want codes.Code
	}{
		{err: domain.ErrVoucherNotFound, want: codes.NotFound},
		{err: domain.ErrIdempotencyConflict, want: codes.AlreadyExists},
		{err: domain.ErrHoldTransition, want: codes.FailedPrecondition},
		{err: domain.ErrMinimumNotMet, want: codes.InvalidArgument},
	} {
		t.Run(tt.err.Error(), func(t *testing.T) {
			if got := status.Code(toStatus(tt.err)); got != tt.want {
				t.Fatalf("toStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}
