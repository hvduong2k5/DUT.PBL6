package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AddressCoordinates là Value Object bất biến đại diện cho tọa độ GPS
type AddressCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// IsValid kiểm tra tọa độ có nằm trong lãnh thổ Việt Nam
func (c AddressCoordinates) IsValid() bool {
	return c.Latitude >= 8.5 && c.Latitude <= 23.5 && c.Longitude >= 102.0 && c.Longitude <= 110.0
}

// ShippingAddress là Entity thuộc Aggregate CustomerProfile (Mô hình 2 cấp hành chính hậu 01/07/2025: Ward -> Province)
type ShippingAddress struct {
	ID            uuid.UUID `json:"id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	RecipientName string    `json:"recipient_name"`
	PhoneNumber   string    `json:"phone_number"`
	StreetAddress string    `json:"street_address"`
	WardCode      string    `json:"ward_code"`
	WardName      string    `json:"ward_name"`
	ProvinceCode  string    `json:"province_code"`
	ProvinceName  string    `json:"province_name"`
	Latitude      *float64  `json:"latitude,omitempty"`
	Longitude     *float64  `json:"longitude,omitempty"`
	Label         string    `json:"label"` // HOME, OFFICE, GIFT_RECIPIENT, OTHER
	IsDefault     bool      `json:"is_default"`
	Version       int       `json:"version"`
	IsDeleted     bool      `json:"is_deleted"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NormalizeAddressLabel converts client-provided labels (Vietnamese or English)
// into valid database enum values: HOME, OFFICE, GIFT_RECIPIENT, OTHER.
func NormalizeAddressLabel(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "HOME", nil
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "HOME", "NHÀ RIÊNG", "NHA RIENG", "NHÀ", "NHA":
		return "HOME", nil
	case "OFFICE", "VĂN PHÒNG", "VAN PHONG", "CƠ QUAN", "CO QUAN", "CÔNG TY", "CONG TY":
		return "OFFICE", nil
	case "GIFT_RECIPIENT", "QUÀ TẶNG", "QUA TANG", "NGƯỜI THÂN", "NGUOI THAN", "GIFT":
		return "GIFT_RECIPIENT", nil
	case "OTHER", "KHÁC", "KHAC":
		return "OTHER", nil
	default:
		return "", fmt.Errorf("invalid label '%s': must be one of HOME (Nhà riêng), OFFICE (Văn phòng), GIFT_RECIPIENT (Quà tặng), OTHER (Khác)", raw)
	}
}
