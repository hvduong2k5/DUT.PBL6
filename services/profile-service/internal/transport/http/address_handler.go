package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/omamx/profile-service/internal/usecase"
)

type AddressHandler struct {
	addressUsecase *usecase.AddressUsecase
}

func NewAddressHandler(addressUsecase *usecase.AddressUsecase) *AddressHandler {
	return &AddressHandler{
		addressUsecase: addressUsecase,
	}
}

type CreateAddressRequest struct {
	RecipientName string   `json:"recipient_name"`
	PhoneNumber   string   `json:"phone_number"`
	StreetAddress string   `json:"street_address"`
	WardCode      string   `json:"ward_code"`
	WardName      string   `json:"ward_name"`
	ProvinceCode  string   `json:"province_code"`
	ProvinceName  string   `json:"province_name"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	Label         string   `json:"label"`
	IsDefault     bool     `json:"is_default"`
}

type ValidateAddressRequest struct {
	RawAddress string `json:"raw_address"`
}

// ListAddresses handles GET /api/v1/profile/addresses
func (h *AddressHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		if paramID := r.URL.Query().Get("customer_id"); paramID != "" {
			if parsed, err := uuid.Parse(paramID); err == nil {
				customerID = parsed
				ok = true
			}
		}
	}
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	addresses, err := h.addressUsecase.ListAddresses(r.Context(), customerID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, addresses)
}

// CreateAddress handles POST /api/v1/profile/addresses
func (h *AddressHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		if paramID := r.URL.Query().Get("customer_id"); paramID != "" {
			if parsed, err := uuid.Parse(paramID); err == nil {
				customerID = parsed
				ok = true
			}
		}
	}
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	var req CreateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	addr := &domain.ShippingAddress{
		CustomerID:    customerID,
		RecipientName: req.RecipientName,
		PhoneNumber:   req.PhoneNumber,
		StreetAddress: req.StreetAddress,
		WardCode:      req.WardCode,
		WardName:      req.WardName,
		ProvinceCode:  req.ProvinceCode,
		ProvinceName:  req.ProvinceName,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Label:         req.Label,
		IsDefault:     req.IsDefault,
	}

	if err := h.addressUsecase.CreateAddress(r.Context(), addr); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, addr)
}

// SwitchDefaultAddress handles PUT /api/v1/profile/addresses/{id}/default
func (h *AddressHandler) SwitchDefaultAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		if paramID := r.URL.Query().Get("customer_id"); paramID != "" {
			if parsed, err := uuid.Parse(paramID); err == nil {
				customerID = parsed
				ok = true
			}
		}
	}
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	addressIDStr := chi.URLParam(r, "id")
	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid address id")
		return
	}

	if err := h.addressUsecase.SwitchDefaultAddress(r.Context(), customerID, addressID); err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			writeJSONError(w, http.StatusNotFound, "address not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":            "default address switched successfully",
		"default_address_id": addressID.String(),
	})
}

// DeleteAddress handles DELETE /api/v1/profile/addresses/{id}
func (h *AddressHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		if paramID := r.URL.Query().Get("customer_id"); paramID != "" {
			if parsed, err := uuid.Parse(paramID); err == nil {
				customerID = parsed
				ok = true
			}
		}
	}
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	addressIDStr := chi.URLParam(r, "id")
	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid address id")
		return
	}

	if err := h.addressUsecase.DeleteAddress(r.Context(), customerID, addressID); err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			writeJSONError(w, http.StatusNotFound, "address not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "address deleted successfully"})
}

// ValidateAddress handles POST /api/v1/profile/addresses/validate
func (h *AddressHandler) ValidateAddress(w http.ResponseWriter, r *http.Request) {
	var req ValidateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RawAddress == "" {
		writeJSONError(w, http.StatusBadRequest, "raw_address is required")
		return
	}

	result, err := h.addressUsecase.ValidateAddress(r.Context(), req.RawAddress)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
