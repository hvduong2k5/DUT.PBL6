package grpcapi

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/omamx/promotion-service/internal/domain"
	commonv1 "github.com/omamx/promotion-service/internal/gen/common/v1"
	promotionv1 "github.com/omamx/promotion-service/internal/gen/promotion/v1"
	"github.com/omamx/promotion-service/internal/repository/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const orderContractVersion = "order-3.2"

type Server struct {
	promotionv1.UnimplementedPromotionServiceServer
	vouchers *postgres.VoucherRepository
}

func NewServer(vouchers *postgres.VoucherRepository) *Server {
	return &Server{vouchers: vouchers}
}

func NewGRPCServer(vouchers *postgres.VoucherRepository, token string) *grpc.Server {
	server := grpc.NewServer(grpc.UnaryInterceptor(authUnaryInterceptor(token)))
	promotionv1.RegisterPromotionServiceServer(server, NewServer(vouchers))
	return server
}

func authUnaryInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		values := metadata.ValueFromIncomingContext(ctx, "authorization")
		provided := ""
		if len(values) > 0 {
			provided = strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer "))
		}
		if token == "" || len(provided) != len(token) || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "valid service authentication is required")
		}
		return handler(ctx, req)
	}
}

func (s *Server) ValidateAndLockVoucher(ctx context.Context, request *promotionv1.ValidateAndLockVoucherRequest) (*promotionv1.ValidateAndLockVoucherResponse, error) {
	if request.GetContractVersion() != orderContractVersion {
		return nil, status.Errorf(codes.FailedPrecondition, "contract_version must be %s", orderContractVersion)
	}
	if request.GetTtlMinutes() < 0 || request.GetTtlMinutes() > 15 {
		return nil, status.Error(codes.InvalidArgument, "ttl_minutes must be between 1 and 15, or 0 for the 15-minute default")
	}
	if err := validateCommand(request.GetIdempotencyKey(), request.GetRequestHash()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	subtotal, currency, err := moneyUnits(request.GetOrderSubtotal())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "order_subtotal must be non-negative VND money")
	}
	if request.GetShippingFee() == nil {
		return nil, status.Error(codes.InvalidArgument, "shipping_fee is required for the voucher contract")
	}
	shippingFee, _, err := moneyUnits(request.GetShippingFee())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "shipping_fee must be non-negative VND money")
	}
	ttl := time.Duration(request.GetTtlMinutes()) * time.Minute
	if request.GetTtlMinutes() == 0 {
		ttl = 15 * time.Minute
	}
	hold, err := s.vouchers.LockVoucher(ctx, postgres.LockRequest{
		IdempotencyKey: request.GetIdempotencyKey(), RequestHash: request.GetRequestHash(),
		VoucherCode: request.GetVoucherCode(), OrderID: request.GetOrderId(), CustomerID: request.GetCustomerId(),
		Subtotal: subtotal, ShippingFee: shippingFee, Currency: currency, TTL: ttl, ContractVersion: request.GetContractVersion(),
	}, time.Now().UTC())
	if err != nil {
		if isEligibilityError(err) {
			return &promotionv1.ValidateAndLockVoucherResponse{IsValid: false, Error: errorDetail(err)}, nil
		}
		return nil, toStatus(err)
	}
	if hold.CommandResult == "REJECTED" {
		return &promotionv1.ValidateAndLockVoucherResponse{
			IsValid: false,
			Error:   lockRejectionDetail(hold.ReasonCode),
			Outcome: outcome(hold, request.GetIdempotencyKey(), "ValidateAndLockVoucher", request.GetRequestHash(), request.GetContractVersion()),
		}, nil
	}
	lockID := hold.ID
	return &promotionv1.ValidateAndLockVoucherResponse{
		IsValid: true, DiscountAmount: money(hold.DiscountAmount, hold.Currency),
		DiscountType: discountType(hold.DiscountType), VoucherLockId: &lockID,
		ExpiresAt: timestamppb.New(hold.ExpiresAt),
		Outcome:   outcome(hold, request.GetIdempotencyKey(), "ValidateAndLockVoucher", request.GetRequestHash(), request.GetContractVersion()),
	}, nil
}

func (s *Server) ReleaseVoucher(ctx context.Context, request *promotionv1.ReleaseVoucherRequest) (*promotionv1.ReleaseVoucherResponse, error) {
	if request.GetIdempotencyKey() == "" || request.GetOrderId() == "" || request.GetReleaseReason() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key, order_id and release_reason are required")
	}
	hold, err := s.vouchers.ReleaseHold(ctx, request.GetIdempotencyKey(), "", request.GetOrderId(), request.GetVoucherLockId(), request.GetVoucherCode(), request.GetReleaseReason(), "", nil, time.Now().UTC())
	if err != nil {
		return nil, toStatus(err)
	}
	return &promotionv1.ReleaseVoucherResponse{IsReleased: hold.State == "RELEASED", Message: hold.State}, nil
}

func (s *Server) CalculateDiscount(ctx context.Context, request *promotionv1.CalculateDiscountRequest) (*promotionv1.CalculateDiscountResponse, error) {
	subtotal, currency, err := moneyUnits(request.GetOrderSubtotal())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "order_subtotal must be non-negative VND money")
	}
	if request.GetShippingFee() == nil {
		return nil, status.Error(codes.InvalidArgument, "shipping_fee is required for the voucher contract")
	}
	shippingFee, _, err := moneyUnits(request.GetShippingFee())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "shipping_fee must be non-negative VND money")
	}
	coupon, discount, err := s.vouchers.CalculateDiscount(ctx, request.GetVoucherCode(), request.GetCustomerId(), subtotal, shippingFee, currency, time.Now().UTC())
	if err != nil {
		return &promotionv1.CalculateDiscountResponse{IsApplicable: false, Description: err.Error()}, nil
	}
	return &promotionv1.CalculateDiscountResponse{IsApplicable: true, EstimatedDiscount: money(discount, currency), Description: fmt.Sprintf("%s discount", coupon.DiscountType)}, nil
}

func (s *Server) GetVoucherHoldByOrder(ctx context.Context, request *promotionv1.VoucherHoldCommand) (*promotionv1.VoucherHoldOutcome, error) {
	if err := validateOrderQuery(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if request.GetExpectedHoldId() != "" && !holdIDPattern.MatchString(request.GetExpectedHoldId()) {
		return nil, status.Error(codes.InvalidArgument, "expected_hold_id must be a UUID")
	}
	hold, proof, err := s.vouchers.GetHoldByOrder(ctx, request.GetOrderId(), request.GetExpectedHoldId(), request.GetQueryAction(), request.GetIdempotencyKey(), request.GetRequestHash())
	if err != nil {
		return nil, toStatus(err)
	}
	response := outcome(hold, "", "", "", request.GetContractVersion())
	if proof != nil {
		response.Command = &commonv1.CommandEvidence{
			IdempotencyKey: proof.IdempotencyKey, Action: proof.Action, RequestHash: proof.RequestHash,
			Result: proof.Result, ReasonCode: proof.ReasonCode, DecisionId: proof.DecisionID,
			DecisionAt: timestamppb.New(proof.DecisionAt), ResultingState: proof.ResultingState,
		}
	}
	return response, nil
}

func (s *Server) FinalizeVoucherHold(ctx context.Context, request *promotionv1.VoucherHoldCommand) (*promotionv1.VoucherHoldOutcome, error) {
	if err := validateOrderCommand(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if request.GetExpectedHoldId() == "" || !holdIDPattern.MatchString(request.GetExpectedHoldId()) || request.ExpectedVersion == nil || request.GetExpectedVersion() < 1 {
		return nil, status.Error(codes.InvalidArgument, "expected_hold_id and positive expected_version are required")
	}
	var expectedVersion *int64
	if request.ExpectedVersion != nil {
		expectedVersion = request.ExpectedVersion
	}
	hold, err := s.vouchers.FinalizeHold(ctx, request.GetIdempotencyKey(), request.GetRequestHash(), request.GetOrderId(), request.GetExpectedHoldId(), expectedVersion, time.Now().UTC())
	if err != nil {
		return nil, toStatus(err)
	}
	return outcome(hold, request.GetIdempotencyKey(), "FinalizeVoucherHold", request.GetRequestHash(), request.GetContractVersion()), nil
}

func (s *Server) ReleaseVoucherHoldByOrder(ctx context.Context, request *promotionv1.VoucherHoldCommand) (*promotionv1.VoucherHoldOutcome, error) {
	if err := validateOrderCommand(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var expectedVersion *int64
	if request.ExpectedVersion != nil {
		expectedVersion = request.ExpectedVersion
	}
	if request.GetExpectedHoldId() == "" || !holdIDPattern.MatchString(request.GetExpectedHoldId()) || expectedVersion == nil || *expectedVersion < 1 {
		return nil, status.Error(codes.InvalidArgument, "expected_hold_id and positive expected_version are required")
	}
	hold, err := s.vouchers.ReleaseHold(ctx, request.GetIdempotencyKey(), request.GetRequestHash(), request.GetOrderId(), request.GetExpectedHoldId(), "", request.GetQueryAction(), request.GetContractVersion(), expectedVersion, time.Now().UTC())
	if err != nil {
		return nil, toStatus(err)
	}
	return outcome(hold, request.GetIdempotencyKey(), "ReleaseVoucherHoldByOrder", request.GetRequestHash(), request.GetContractVersion()), nil
}

func validateOrderCommand(request *promotionv1.VoucherHoldCommand) error {
	if request.GetContractVersion() != orderContractVersion {
		return fmt.Errorf("contract_version must be %s", orderContractVersion)
	}
	if request.GetOrderId() == "" || request.GetQueryAction() == "" {
		return errors.New("order_id and query_action are required")
	}
	return validateCommand(request.GetIdempotencyKey(), request.GetRequestHash())
}

func validateOrderQuery(request *promotionv1.VoucherHoldCommand) error {
	if request.GetContractVersion() != orderContractVersion {
		return fmt.Errorf("contract_version must be %s", orderContractVersion)
	}
	if request.GetOrderId() == "" {
		return errors.New("order_id is required")
	}
	if request.GetQueryAction() == "" {
		if request.GetIdempotencyKey() != "" || request.GetRequestHash() != "" {
			return errors.New("query_action, idempotency_key and request_hash must be provided together")
		}
		return nil
	}
	if request.GetQueryAction() != "ValidateAndLockVoucher" && request.GetQueryAction() != "FinalizeVoucherHold" && request.GetQueryAction() != "ReleaseVoucherHoldByOrder" {
		return errors.New("query_action must name a supported voucher command")
	}
	return validateCommand(request.GetIdempotencyKey(), request.GetRequestHash())
}

var idempotencyKeyPattern = regexp.MustCompile(`^[\x21-\x7e]{16,128}$`)
var holdIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validateCommand(key, requestHash string) error {
	if !idempotencyKeyPattern.MatchString(key) {
		return errors.New("idempotency_key must contain 16 to 128 visible ASCII characters")
	}
	if len(requestHash) != 64 {
		return errors.New("request_hash must be a SHA-256 hex digest")
	}
	decoded, err := hex.DecodeString(requestHash)
	if err != nil || len(decoded) != 32 || strings.ToLower(requestHash) != requestHash {
		return errors.New("request_hash must be lowercase SHA-256 hex")
	}
	return nil
}

func moneyUnits(value *commonv1.Money) (int64, string, error) {
	if value == nil || value.GetCurrencyCode() != "VND" || value.GetNanos() != 0 || value.GetUnits() < 0 {
		return 0, "", errors.New("invalid VND money")
	}
	return value.GetUnits(), value.GetCurrencyCode(), nil
}

func money(units int64, currency string) *commonv1.Money {
	return &commonv1.Money{CurrencyCode: currency, Units: units}
}

func discountType(value domain.DiscountType) promotionv1.DiscountType {
	switch value {
	case domain.DiscountFixedAmount:
		return promotionv1.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT
	case domain.DiscountPercentage:
		return promotionv1.DiscountType_DISCOUNT_TYPE_PERCENTAGE
	case domain.DiscountFreeShipping:
		return promotionv1.DiscountType_DISCOUNT_TYPE_FREE_SHIPPING
	default:
		return promotionv1.DiscountType_DISCOUNT_TYPE_UNSPECIFIED
	}
}

func outcome(hold postgres.Hold, key, action, requestHash, contractVersion string) *promotionv1.VoucherHoldOutcome {
	observed := hold.ObservedAt
	if observed.IsZero() {
		observed = time.Now().UTC()
	}
	result := &promotionv1.VoucherHoldOutcome{
		OrderId: hold.OrderID, HoldId: hold.ID, State: hold.State, ResourceVersion: hold.Version,
		ObservedAt:            timestamppb.New(observed),
		CancellationTombstone: hold.Tombstone, DiscountAmount: money(hold.DiscountAmount, hold.Currency), ContractVersion: contractVersion,
	}
	if action != "" {
		decisionAt := hold.DecisionAt
		if decisionAt.IsZero() {
			decisionAt = observed
		}
		resultingState := hold.ResultingState
		if resultingState == "" {
			resultingState = hold.State
		}
		result.Command = &commonv1.CommandEvidence{IdempotencyKey: key, Action: action, RequestHash: requestHash,
			Result: commandResult(hold), ReasonCode: hold.ReasonCode, DecisionId: decisionID(hold),
			DecisionAt: timestamppb.New(decisionAt), ResultingState: resultingState}
	}
	if !hold.Tombstone && !hold.ExpiresAt.IsZero() {
		result.ExpiresAt = timestamppb.New(hold.ExpiresAt)
	}
	return result
}

func decisionID(hold postgres.Hold) string {
	if hold.DecisionID != "" {
		return hold.DecisionID
	}
	return hold.ID
}

func commandResult(hold postgres.Hold) string {
	if hold.CommandResult == "REJECTED" {
		return "REJECTED"
	}
	if hold.Replay {
		return "ALREADY_APPLIED"
	}
	if hold.CommandResult != "" {
		return hold.CommandResult
	}
	return "APPLIED"
}

func isEligibilityError(err error) bool {
	return errors.Is(err, domain.ErrVoucherNotFound) || errors.Is(err, domain.ErrVoucherInactive) ||
		errors.Is(err, domain.ErrVoucherExpired) || errors.Is(err, domain.ErrMinimumNotMet) ||
		errors.Is(err, domain.ErrUsageLimit) || errors.Is(err, domain.ErrVoucherAlreadyLocked)
}

func errorDetail(err error) *commonv1.ErrorDetail {
	code := commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR
	switch {
	case errors.Is(err, domain.ErrVoucherNotFound):
		code = commonv1.ErrorCode_ERR_PROMOTION_VOUCHER_NOT_FOUND
	case errors.Is(err, domain.ErrVoucherExpired), errors.Is(err, domain.ErrVoucherInactive):
		code = commonv1.ErrorCode_ERR_PROMOTION_VOUCHER_EXPIRED
	case errors.Is(err, domain.ErrMinimumNotMet):
		code = commonv1.ErrorCode_ERR_PROMOTION_MIN_ORDER_AMOUNT_NOT_MET
	case errors.Is(err, domain.ErrUsageLimit):
		code = commonv1.ErrorCode_ERR_PROMOTION_USAGE_LIMIT_EXCEEDED
	case errors.Is(err, domain.ErrVoucherAlreadyLocked):
		code = commonv1.ErrorCode_ERR_PROMOTION_VOUCHER_ALREADY_LOCKED
	}
	return &commonv1.ErrorDetail{ErrorCode: code, UserMessage: err.Error(), DevMessage: err.Error(), Domain: "promotion"}
}

func lockRejectionDetail(reasonCode string) *commonv1.ErrorDetail {
	code := commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR
	message := "voucher cannot be applied"
	switch reasonCode {
	case "VOUCHER_NOT_FOUND":
		code, message = commonv1.ErrorCode_ERR_PROMOTION_VOUCHER_NOT_FOUND, "voucher was not found"
	case "VOUCHER_INACTIVE", "VOUCHER_EXPIRED":
		code, message = commonv1.ErrorCode_ERR_PROMOTION_VOUCHER_EXPIRED, "voucher is not active"
	case "MINIMUM_ORDER_AMOUNT_NOT_MET":
		code, message = commonv1.ErrorCode_ERR_PROMOTION_MIN_ORDER_AMOUNT_NOT_MET, "minimum order amount was not met"
	case "USAGE_LIMIT_EXCEEDED":
		code, message = commonv1.ErrorCode_ERR_PROMOTION_USAGE_LIMIT_EXCEEDED, "voucher usage limit was reached"
	case "VOUCHER_ALREADY_LOCKED":
		code, message = commonv1.ErrorCode_ERR_PROMOTION_VOUCHER_ALREADY_LOCKED, "voucher is already locked for this order"
	}
	return &commonv1.ErrorDetail{ErrorCode: code, UserMessage: message, DevMessage: message, Domain: "promotion"}
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrVoucherNotFound), errors.Is(err, domain.ErrHoldNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrIdempotencyConflict), errors.Is(err, domain.ErrVoucherAlreadyLocked):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrHoldTransition), errors.Is(err, domain.ErrVoucherExpired), errors.Is(err, domain.ErrUsageLimit):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrHoldVersionConflict):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrMinimumNotMet), errors.Is(err, domain.ErrVoucherInactive):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "promotion operation failed")
	}
}

func Serve(listener net.Listener, server *grpc.Server) error { return server.Serve(listener) }
