package shippingprovider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	api "github.com/omamx/order-service/internal/gen/shipping/v2"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Quote struct {
	ID, Carrier, Service string
	Version, Fee         int64
	Until                time.Time
}

// This profile uses the same business-field JSON codec as Shipping v2. The live
// integration exercises the provider's independent canonical-hash verifier.
func quoteHash(message protoreflect.Message) (string, error) {
	var value func(protoreflect.Message) (map[string]any, error)
	value = func(message protoreflect.Message) (map[string]any, error) {
		result := map[string]any{}
		fields := message.Descriptor().Fields()
		for index := 0; index < fields.Len(); index++ {
			field := fields.Get(index)
			name := string(field.Name())
			if name == "request_hash" || name == "idempotency_key" || name == "correlation_id" || !message.Has(field) {
				continue
			}
			item := message.Get(field)
			switch field.Kind() {
			case protoreflect.MessageKind:
				nested, err := value(item.Message())
				if err != nil {
					return nil, err
				}
				if len(nested) > 0 {
					result[name] = nested
				}
			case protoreflect.StringKind:
				result[name] = item.String()
			case protoreflect.EnumKind:
				result[name] = int32(item.Enum())
			case protoreflect.BoolKind:
				result[name] = item.Bool()
			case protoreflect.Int32Kind, protoreflect.Int64Kind:
				result[name] = item.Int()
			default:
				return nil, errors.New("unsupported shipping quote field")
			}
		}
		return result, nil
	}
	data, err := value(message)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(data); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(strings.TrimSuffix(buffer.String(), "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func (a *Adapter) Quote(ctx context.Context, address domain.Address, weight int64, carrier, service, warehouse, key string) (Quote, error) {
	var quote Quote
	if a == nil || weight < 1 {
		return quote, domain.E("SHIPPING_QUOTE_UNAVAILABLE", 503)
	}
	request := &api.CalculateShippingFeeRequest{IdempotencyKey: key, ContractVersion: "order-3.2", DestinationAddress: &common.Address{RecipientName: address.Name, PhoneNumber: address.Phone, StreetAddress: address.Street, Ward: address.Ward, Province: address.Province, CountryCode: "VN"}, TotalWeight: &common.Weight{DecimalValue: strconv.FormatInt(weight, 10), Unit: common.WeightUnit_WEIGHT_UNIT_GRAM}, PreferredCarrier: &carrier, ServiceCode: &service, OriginWarehouseId: &warehouse}
	hash, err := quoteHash(request.ProtoReflect())
	if err != nil {
		return quote, err
	}
	request.RequestHash = hash
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.Token)
	response, err := a.Client.CalculateShippingFee(ctx, request)
	if err != nil {
		return quote, domain.E("SHIPPING_QUOTE_UNAVAILABLE", 503)
	}
	if response.GetContractVersion() != "order-3.2" || response.GetRequestHash() != hash || response.GetQuoteVersion() < 1 || response.GetError() != nil || response.GetShippingFee() == nil || response.ShippingFee.CurrencyCode != "VND" || response.ShippingFee.Nanos != 0 || response.ShippingFee.Units < 0 || response.GetQuoteExpiresAt() == nil || response.QuoteExpiresAt.CheckValid() != nil || !time.Now().Before(response.QuoteExpiresAt.AsTime()) || response.GetCarrier() == "" || response.GetServiceCode() == "" {
		return quote, domain.E("INVALID_SHIPPING_QUOTE", 503)
	}
	if id, err := uuid.Parse(response.GetQuoteId()); err != nil || id == uuid.Nil {
		return quote, domain.E("INVALID_SHIPPING_QUOTE", 503)
	}
	return Quote{ID: response.QuoteId, Carrier: response.Carrier, Service: response.ServiceCode, Version: response.QuoteVersion, Fee: response.ShippingFee.Units, Until: response.QuoteExpiresAt.AsTime().Truncate(time.Microsecond)}, nil
}
