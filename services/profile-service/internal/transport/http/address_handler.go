package http

import (
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
	IsDefault     *bool    `json:"is_default,omitempty"`
}

type ValidateAddressRequest struct {
	RawAddress string `json:"raw_address"`
}

// ListAddresses handles GET /api/v1/profile/addresses
func (h *AddressHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	addresses, err := h.addressUsecase.ListAddresses(r.Context(), customerID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, addresses)
}

// CreateAddress handles POST /api/v1/profile/addresses
func (h *AddressHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	var req CreateAddressRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	normLabel, err := domain.NormalizeAddressLabel(req.Label)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
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
		Label:         normLabel,
		IsDefault:     req.IsDefault != nil && *req.IsDefault,
	}

	if err := h.addressUsecase.CreateAddress(r.Context(), addr); err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("ETag", etag(addr.Version))
	w.Header().Set("Location", "/api/v1/profile/addresses/"+addr.ID.String())
	writeJSON(w, http.StatusCreated, addr)
}

// UpdateAddress handles PUT /api/v1/profile/addresses/{id}
func (h *AddressHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
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

	var req CreateAddressRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	normLabel, err := domain.NormalizeAddressLabel(req.Label)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	version, err := expectedVersion(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if req.IsDefault != nil {
		writeDomainError(w, &domain.ValidationError{Field: "is_default", Message: "use the dedicated default endpoint"})
		return
	}
	addr := &domain.ShippingAddress{
		Version:       version,
		ID:            addressID,
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
		Label:         normLabel,
	}

	if err := h.addressUsecase.UpdateAddress(r.Context(), addr); err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			writeJSONError(w, http.StatusNotFound, "address not found")
			return
		}
		writeDomainError(w, err)
		return
	}

	w.Header().Set("ETag", etag(addr.Version))
	writeJSON(w, http.StatusOK, addr)
}

// GetAddress handles GET /api/v1/profile/addresses/{id} and POST simulation requests
func (h *AddressHandler) GetAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, 401, "unauthorized")
		return
	}
	addressIDStr := chi.URLParam(r, "id")
	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid address id")
		return
	}

	addr, err := h.addressUsecase.GetOwnedAddress(r.Context(), customerID, addressID)
	if err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			writeJSONError(w, http.StatusNotFound, "address not found")
			return
		}
		writeDomainError(w, err)
		return
	}

	w.Header().Set("ETag", etag(addr.Version))
	writeJSON(w, http.StatusOK, addr)
}

// SwitchDefaultAddress handles PUT /api/v1/profile/addresses/{id}/default
func (h *AddressHandler) SwitchDefaultAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
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

	version, err := expectedVersion(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := h.addressUsecase.SwitchDefaultAddressWithVersion(r.Context(), customerID, addressID, version); err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			writeJSONError(w, http.StatusNotFound, "address not found")
			return
		}
		writeDomainError(w, err)
		return
	}

	addresses, err := h.addressUsecase.ListAddresses(r.Context(), customerID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, 200, addresses)
}

// DeleteAddress handles DELETE /api/v1/profile/addresses/{id}
func (h *AddressHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
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

	version, err := expectedVersion(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := h.addressUsecase.DeleteAddressWithVersion(r.Context(), customerID, addressID, version); err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			writeJSONError(w, http.StatusNotFound, "address not found")
			return
		}
		writeDomainError(w, err)
		return
	}

	addresses, err := h.addressUsecase.ListAddresses(r.Context(), customerID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, 200, addresses)
}

// ValidateAddress handles POST /api/v1/profile/addresses/validate
func (h *AddressHandler) ValidateAddress(w http.ResponseWriter, r *http.Request) {
	var req ValidateAddressRequest
	if err := decodeBody(w, r, &req); err != nil || req.RawAddress == "" {
		writeJSONError(w, http.StatusBadRequest, "raw_address is required")
		return
	}

	result, err := h.addressUsecase.ValidateAddress(r.Context(), req.RawAddress)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ValidateConsistency handles POST /api/v1/profile/addresses/validate-consistency
func (h *AddressHandler) ValidateConsistency(w http.ResponseWriter, r *http.Request) {
	var req usecase.ConsistencyVerificationRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StreetAddress == "" || req.WardCode == "" {
		writeJSONError(w, http.StatusBadRequest, "street_address and ward_code are required")
		return
	}

	result, err := h.addressUsecase.ValidateAddressConsistency(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
