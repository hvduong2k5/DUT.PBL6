package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/cache"
	"github.com/omamx/profile-service/internal/repository"
)

type AddressUsecase struct {
	addrRepo     *repository.AddressRepository
	fuzzyMatcher *AddressFuzzyMatcher
	cache        *cache.DualLayerCache
	pool         *pgxpool.Pool
}

func NewAddressUsecase(
	addrRepo *repository.AddressRepository,
	fuzzyMatcher *AddressFuzzyMatcher,
	cache *cache.DualLayerCache,
	pool *pgxpool.Pool,
) *AddressUsecase {
	return &AddressUsecase{
		addrRepo:     addrRepo,
		fuzzyMatcher: fuzzyMatcher,
		cache:        cache,
		pool:         pool,
	}
}

// ListAddresses returns non-deleted addresses for customer.
func (u *AddressUsecase) ListAddresses(ctx context.Context, customerID uuid.UUID) ([]*domain.ShippingAddress, error) {
	return u.addrRepo.ListAddressesByCustomerID(ctx, customerID)
}

// CreateAddress saves a new shipping address.
func (u *AddressUsecase) CreateAddress(ctx context.Context, addr *domain.ShippingAddress) error {
	now := time.Now().UTC()
	if addr.ID == uuid.Nil {
		addr.ID = uuid.New()
	}
	addr.CreatedAt = now
	addr.UpdatedAt = now

	// If this is the customer's first address, make it default automatically
	count, err := u.addrRepo.CountDefaultAddresses(ctx, addr.CustomerID)
	if err == nil && count == 0 {
		addr.IsDefault = true
	}

	if err := u.addrRepo.CreateAddress(ctx, addr); err != nil {
		return err
	}

	// Invalidate customer cache
	_ = u.cache.Invalidate(ctx, addr.CustomerID.String())
	return nil
}

// SwitchDefaultAddress atomically sets an address as default and purges dual-layer cache.
func (u *AddressUsecase) SwitchDefaultAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	if err := u.addrRepo.SwitchDefaultAddress(ctx, customerID, addressID); err != nil {
		return err
	}

	// Purge local L1 + remote Redis and broadcast invalidation to prevent stale address read
	_ = u.cache.Invalidate(ctx, customerID.String())
	return nil
}

// DeleteAddress soft-deletes address and purges cache.
func (u *AddressUsecase) DeleteAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	if err := u.addrRepo.DeleteAddress(ctx, customerID, addressID); err != nil {
		return err
	}
	_ = u.cache.Invalidate(ctx, customerID.String())
	return nil
}

// ValidateAddress runs the 2-Stage Fuzzy Matching pipeline against administrative units.
// Note: This is an Async/UI path and is STRICTLY excluded from the Checkout Critical Path.
func (u *AddressUsecase) ValidateAddress(ctx context.Context, userInput string) (*MatchResult, error) {
	// Stage 1: Pre-filter top candidates from DB using pg_trgm similarity
	query := `
		SELECT code, name, level
		FROM administrative_units
		WHERE level = 'WARD'
		  AND name % $1
		ORDER BY similarity(name, $1) DESC
		LIMIT 20
	`
	rows, err := u.pool.Query(ctx, query, userInput)
	if err != nil {
		return nil, fmt.Errorf("failed querying administrative candidates: %w", err)
	}
	defer rows.Close()

	var candidates []AdministrativeCandidate
	for rows.Next() {
		var c AdministrativeCandidate
		if err := rows.Scan(&c.Code, &c.Name, &c.Level); err == nil {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		// Fallback: If no direct trigram match, select active wards of Thừa Thiên Huế for in-memory Levenshtein
		fallbackQuery := `SELECT code, name, level FROM administrative_units WHERE level = 'WARD' LIMIT 50`
		fbRows, fbErr := u.pool.Query(ctx, fallbackQuery)
		if fbErr == nil {
			defer fbRows.Close()
			for fbRows.Next() {
				var c AdministrativeCandidate
				if err := fbRows.Scan(&c.Code, &c.Name, &c.Level); err == nil {
					candidates = append(candidates, c)
				}
			}
		}
	}

	// Stage 2: Refine similarity with Vietnamese diacritics stripping & Levenshtein
	result := u.fuzzyMatcher.RefineCandidates(userInput, candidates)
	if result == nil {
		return nil, fmt.Errorf("no matching administrative ward found for input: %s", userInput)
	}

	return result, nil
}

// GetDeliveryAddressCheckout fetches delivery address on Checkout Critical Path ($P99 <= 5ms).
// Bypasses fuzzy matching; uses direct primary key / default B-Tree lookup.
func (u *AddressUsecase) GetDeliveryAddressCheckout(ctx context.Context, addressID, customerID uuid.UUID) (*domain.ShippingAddress, error) {
	if addressID != uuid.Nil {
		return u.addrRepo.GetAddressByID(ctx, addressID)
	}
	return u.addrRepo.GetDefaultAddress(ctx, customerID)
}

// ConsistencyVerificationRequest DTO for cross-field address validation
type ConsistencyVerificationRequest struct {
	StreetAddress string `json:"street_address"`
	WardCode      string `json:"ward_code"`
	ProvinceCode  string `json:"province_code"`
}

type WardReference struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

type ConsistencyCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type AddressConsistencyResponse struct {
	IsConsistent    bool                    `json:"is_consistent"`
	ConfidenceScore float64                 `json:"confidence_score"`
	WarningLevel    string                  `json:"warning_level"` // "VERIFIED", "MISMATCH_DETECTED", "MULTI_WARD_STREET", "UNVERIFIED_NEW_STREET"
	DetectedStreet  string                  `json:"detected_street"`
	SelectedWard    WardReference           `json:"selected_ward"`
	SuggestedWard   *WardReference          `json:"suggested_ward,omitempty"`
	Coordinates     *ConsistencyCoordinates `json:"coordinates,omitempty"`
	Message         string                  `json:"message"`
}

// ExtractStreetName extracts the street component by stripping house numbers, alley prefixes, etc.
func ExtractStreetName(raw string) string {
	cleaned := strings.TrimSpace(raw)
	// Regex removing house numbers, kiệt, ngõ, hẻm, số at the beginning
	re := regexp.MustCompile(`^(?i)(?:nhà\s*số\s*|số\s*|kiệt\s*\d+[a-zA-Z\/\d\-\.]*\s*|ngõ\s*\d+[a-zA-Z\/\d\-\.]*\s*|hẻm\s*\d+[a-zA-Z\/\d\-\.]*\s*|\d+[a-zA-Z\/\d\-\.]*[\s,]+)*`)
	cleaned = re.ReplaceAllString(cleaned, "")
	cleaned = strings.Trim(cleaned, " ,.-")
	if cleaned == "" {
		return raw
	}
	return cleaned
}

// ValidateAddressConsistency verifies whether the typed street address matches the selected ward dropdown.
func (u *AddressUsecase) ValidateAddressConsistency(ctx context.Context, req ConsistencyVerificationRequest) (*AddressConsistencyResponse, error) {
	if strings.TrimSpace(req.StreetAddress) == "" {
		return nil, fmt.Errorf("street_address cannot be empty")
	}

	detectedStreet := ExtractStreetName(req.StreetAddress)
	unaccentedStreet := strings.ToLower(StripVietnameseTones(detectedStreet))

	// Fetch official ward name for the user's selected dropdown
	selectedWardName, err := u.addrRepo.GetWardName(ctx, req.WardCode)
	if err != nil || selectedWardName == "" {
		selectedWardName = req.WardCode
	}

	selectedWardRef := WardReference{
		Code: req.WardCode,
		Name: selectedWardName,
	}

	// Tra cứu CSDL tuyến đường (hỗ trợ toàn quốc)
	mappings, err := u.addrRepo.FindStreetMappings(ctx, req.ProvinceCode, unaccentedStreet)
	if err != nil || len(mappings) == 0 {
		// Fallback: Tuyến đường chưa có trong từ điển định tuyến nhanh (ngõ hẻm sâu hoặc tỉnh thành khác)
		// -> KHÔNG block người dùng, trả về UNVERIFIED_NEW_STREET với is_consistent = true
		return &AddressConsistencyResponse{
			IsConsistent:    true,
			ConfidenceScore: 0.70,
			WarningLevel:    "UNVERIFIED_NEW_STREET",
			DetectedStreet:  detectedStreet,
			SelectedWard:    selectedWardRef,
			Message:         fmt.Sprintf("Tuyến đường '%s' chưa có trong danh mục định tuyến nhanh. Hệ thống sẽ giao hàng theo số nhà và phường/xã '%s' đã chọn.", detectedStreet, selectedWardName),
		}, nil
	}

	// Kiểm tra xem ward_code của người dùng có nằm trong các phường mà con đường đi qua không
	var matchedMapping *repository.StreetWardMapping
	for _, m := range mappings {
		if m.WardCode == req.WardCode {
			mappingCopy := m
			matchedMapping = &mappingCopy
			break
		}
	}

	// TRƯỜNG HỢP 1: HOÀN TOÀN KHỚP
	if matchedMapping != nil {
		var coords *ConsistencyCoordinates
		if matchedMapping.Latitude != nil && matchedMapping.Longitude != nil {
			coords = &ConsistencyCoordinates{
				Latitude:  *matchedMapping.Latitude,
				Longitude: *matchedMapping.Longitude,
			}
		}

		warningLevel := "VERIFIED"
		message := fmt.Sprintf("Địa chỉ hợp lệ. Tuyến đường '%s' thuộc '%s'.", detectedStreet, selectedWardName)
		if len(mappings) > 1 {
			warningLevel = "MULTI_WARD_STREET"
			message = fmt.Sprintf("Tuyến đường '%s' trải dài qua nhiều phường. Lựa chọn '%s' của bạn hoàn toàn hợp lệ.", detectedStreet, selectedWardName)
		}

		return &AddressConsistencyResponse{
			IsConsistent:    true,
			ConfidenceScore: 1.0,
			WarningLevel:    warningLevel,
			DetectedStreet:  detectedStreet,
			SelectedWard:    selectedWardRef,
			Coordinates:     coords,
			Message:         message,
		}, nil
	}

	// TRƯỜNG HỢP 2: PHÁT HIỆN MÂU THUẪN (MISMATCH DETECTED)
	primaryMapping := mappings[0]
	for _, m := range mappings {
		if m.IsPrimary {
			primaryMapping = m
			break
		}
	}

	var coords *ConsistencyCoordinates
	if primaryMapping.Latitude != nil && primaryMapping.Longitude != nil {
		coords = &ConsistencyCoordinates{
			Latitude:  *primaryMapping.Latitude,
			Longitude: *primaryMapping.Longitude,
		}
	}

	suggestedWard := &WardReference{
		Code:   primaryMapping.WardCode,
		Name:   primaryMapping.WardName,
		Reason: fmt.Sprintf("Tuyến đường '%s' thuộc %s, không thuộc %s.", detectedStreet, primaryMapping.WardName, selectedWardName),
	}

	return &AddressConsistencyResponse{
		IsConsistent:    false,
		ConfidenceScore: 0.95,
		WarningLevel:    "MISMATCH_DETECTED",
		DetectedStreet:  detectedStreet,
		SelectedWard:    selectedWardRef,
		SuggestedWard:   suggestedWard,
		Coordinates:     coords,
		Message:         fmt.Sprintf("Phát hiện mâu thuẫn: Tuyến đường '%s' thuộc %s. Bạn có muốn đổi sang %s không?", detectedStreet, primaryMapping.WardName, primaryMapping.WardName),
	}, nil
}
