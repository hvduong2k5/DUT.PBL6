package domain

import (
	"time"

	"github.com/google/uuid"
)

// CustomerStatus đại diện cho trạng thái hồ sơ khách hàng theo Customer Profile FSM
type CustomerStatus string

const (
	CustomerStatusActive              CustomerStatus = "ACTIVE"
	CustomerStatusSuspended           CustomerStatus = "SUSPENDED"
	CustomerStatusPendingVerification CustomerStatus = "PENDING_VERIFICATION"
)

// GenderEnum đại diện cho giới tính người dùng
type GenderEnum string

const (
	GenderMale        GenderEnum = "MALE"
	GenderFemale      GenderEnum = "FEMALE"
	GenderOther       GenderEnum = "OTHER"
	GenderUnspecified GenderEnum = "UNSPECIFIED"
)

// DietaryPreference là Value Object quản lý sở thích ẩm thực OCOP và cảnh báo dị ứng
type DietaryPreference struct {
	FavoriteProducts  []string `json:"favorite_products"`
	DietaryPreference string   `json:"dietary_preference"` // NORMAL, LOW_SUGAR, VEGAN
	AllergyAlert      []string `json:"allergy_alert"`      // PEANUT, SESAME, GLUTEN
}

// CustomerProfile là Aggregate Root của BC-15 User & Customer Profile Context
type CustomerProfile struct {
	ID          uuid.UUID         `json:"id"`
	UserID      uuid.UUID         `json:"user_id"`
	FullName    string            `json:"full_name"`
	PhoneNumber string            `json:"phone_number"`
	Email       string            `json:"email"`
	DateOfBirth *time.Time        `json:"date_of_birth,omitempty"`
	Gender      string            `json:"gender"`
	AvatarURL   string            `json:"avatar_url"`
	Preferences map[string]any    `json:"preferences"`
	Status      string            `json:"status"`
	Version     int               `json:"version"` // Khóa lạc quan Non-blocking CAS
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// GetDietaryPreference trích xuất DietaryPreference strongly typed từ map Preferences
func (c *CustomerProfile) GetDietaryPreference() DietaryPreference {
	var dp DietaryPreference
	if c.Preferences == nil {
		dp.DietaryPreference = "NORMAL"
		return dp
	}

	if val, ok := c.Preferences["dietary_preference"].(string); ok {
		dp.DietaryPreference = val
	} else {
		dp.DietaryPreference = "NORMAL"
	}

	if prods, ok := c.Preferences["favorite_products"].([]any); ok {
		for _, p := range prods {
			if str, ok := p.(string); ok {
				dp.FavoriteProducts = append(dp.FavoriteProducts, str)
			}
		}
	}

	if alerts, ok := c.Preferences["allergy_alert"].([]any); ok {
		for _, a := range alerts {
			if str, ok := a.(string); ok {
				dp.AllergyAlert = append(dp.AllergyAlert, str)
			}
		}
	}

	return dp
}
