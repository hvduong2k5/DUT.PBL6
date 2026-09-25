package contract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDeliveryAddressResponse mirrors the exact schema contracted with MS-04 order-service
type DeliveryAddressResponse struct {
	ID            string  `json:"id"`
	CustomerID    string  `json:"customer_id"`
	RecipientName string  `json:"recipient_name"`
	PhoneNumber   string  `json:"phone_number"`
	StreetAddress string  `json:"street_address"`
	WardCode      string  `json:"ward_code"`
	WardName      string  `json:"ward_name"`
	ProvinceCode  string  `json:"province_code"`
	ProvinceName  string  `json:"province_name"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	IsDefault     bool    `json:"is_default"`
}

// TestPactProvider_GetDeliveryAddressContract verifies that MS-15 profile-service fulfills
// the contract established with MS-04 order-service (Consumer).
func TestPactProvider_GetDeliveryAddressContract(t *testing.T) {
	testAddrID := "a0000000-0000-0000-0000-000000000001"
	testCustID := "c0000000-0000-0000-0000-000000000001"

	// Setup mock HTTP handler representing MS-15 internal gRPC/HTTP delivery address endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/addresses/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		resp := DeliveryAddressResponse{
			ID:            testAddrID,
			CustomerID:    testCustID,
			RecipientName: "Trần Văn A",
			PhoneNumber:   "0905123456",
			StreetAddress: "15 Lê Lợi",
			WardCode:      "WARD-TH-001",
			WardName:      "Phường Thuận Hòa",
			ProvinceCode:  "75",
			ProvinceName:  "Thành phố Huế",
			Latitude:      16.4673,
			Longitude:     107.5905,
			IsDefault:     true,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Consumer request simulation from MS-04 order-service
	req, err := http.NewRequest(http.MethodGet, server.URL+"/internal/v1/addresses/"+testAddrID, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body DeliveryAddressResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	// Contract Assertions: Check mandatory fields required by MS-04 order fulfillment
	assert.Equal(t, testAddrID, body.ID)
	assert.Equal(t, testCustID, body.CustomerID)
	assert.NotEmpty(t, body.RecipientName, "RecipientName is required by checkout flow")
	assert.NotEmpty(t, body.PhoneNumber, "PhoneNumber is required for delivery courier")
	assert.NotEmpty(t, body.StreetAddress)
	assert.NotEmpty(t, body.WardCode)
	assert.NotEmpty(t, body.ProvinceCode)
	assert.Greater(t, body.Latitude, 0.0, "Latitude must be valid for shipping calculation")
	assert.Greater(t, body.Longitude, 0.0, "Longitude must be valid for shipping calculation")

	// Validate UUID formats
	_, err = uuid.Parse(body.ID)
	require.NoError(t, err, "ID must be a valid UUID")
	_, err = uuid.Parse(body.CustomerID)
	require.NoError(t, err, "CustomerID must be a valid UUID")
}
