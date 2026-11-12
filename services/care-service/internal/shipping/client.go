// Package shipping contains Care's trusted service-to-service Shipping adapter.
package shipping

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/omamx/care-service/internal/domain"
)

const (
	createReturnShipmentPath = "/api/v1/internal/return-shipments"
	returnCarrierCode        = "FAKE"
	returnServiceCode        = "STANDARD"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// ApprovedDecision is the minimum, Care-owned decision context required to
// request a reverse pickup. Callers should construct it from a persisted,
// immutable decision, never directly from a browser payload.
type ApprovedDecision struct {
	CaseID          string
	DecisionVersion int
	Status          domain.ReturnCaseStatus
	ReturnRequired  bool
	OrderID         string
	PackageID       string
	ReasonCode      string
	PickupAddress   Address
	Destination     Address
	Parcel          Parcel
}

type Address struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Street      string `json:"street"`
	Ward        string `json:"ward"`
	District    string `json:"district,omitempty"`
	Province    string `json:"province"`
	CountryCode string `json:"country_code,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
}

type Parcel struct {
	WeightGrams int64 `json:"weight_grams"`
	LengthCM    int64 `json:"length_cm"`
	WidthCM     int64 `json:"width_cm"`
	HeightCM    int64 `json:"height_cm"`
	Fragile     bool  `json:"fragile,omitempty"`
}

type ReturnShipment struct {
	ID                      string  `json:"return_shipment_id"`
	ShipmentNumber          string  `json:"shipment_number"`
	ReturnCaseID            string  `json:"return_case_id"`
	OrderID                 string  `json:"order_id"`
	PackageID               string  `json:"package_id"`
	CarrierCode             string  `json:"carrier_code"`
	ServiceCode             string  `json:"service_code"`
	CarrierOrderID          string  `json:"carrier_order_id,omitempty"`
	TrackingCode            string  `json:"tracking_code,omitempty"`
	Status                  string  `json:"status"`
	PickupAddress           Address `json:"pickup_address"`
	Destination             Address `json:"destination"`
	Parcel                  Parcel  `json:"parcel"`
	CODAmountVND            int64   `json:"cod_amount_vnd"`
	FeePayer                string  `json:"fee_payer"`
	ProviderEstimatedFeeVND int64   `json:"provider_estimated_fee_vnd"`
	ReasonCode              string  `json:"reason_code"`
	FailureCode             string  `json:"failure_code,omitempty"`
	Revision                int64   `json:"revision"`
}

type Error struct {
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("shipping returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("shipping returned HTTP %d (%s)", e.StatusCode, e.Code)
}

func NewClient(baseURL, token string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("shipping base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("shipping internal token is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(parsed.String(), "/"), token: strings.TrimSpace(token), http: httpClient}, nil
}

// CreateForApprovedDecision is intentionally the only exposed operation: an
// approved decision requiring a physical return is a precondition to contacting
// Shipping. The idempotency key is stable across retries of this decision.
func (c *Client) CreateForApprovedDecision(ctx context.Context, decision ApprovedDecision) (ReturnShipment, error) {
	if decision.Status != domain.ReturnCaseApproved && decision.Status != domain.ReturnCasePartiallyApproved {
		return ReturnShipment{}, errors.New("return shipment requires an approved Care decision")
	}
	if !decision.ReturnRequired {
		return ReturnShipment{}, errors.New("approved Care decision does not require returned goods")
	}
	decision.CaseID = strings.TrimSpace(decision.CaseID)
	decision.OrderID = strings.TrimSpace(decision.OrderID)
	decision.PackageID = strings.TrimSpace(decision.PackageID)
	decision.ReasonCode = strings.TrimSpace(decision.ReasonCode)
	if decision.CaseID == "" || decision.OrderID == "" || decision.PackageID == "" || decision.ReasonCode == "" || decision.DecisionVersion < 1 {
		return ReturnShipment{}, errors.New("approved Care decision is missing its case, order, package, reason, or version")
	}
	requestBody := struct {
		ReturnCaseID  string  `json:"return_case_id"`
		OrderID       string  `json:"order_id"`
		PackageID     string  `json:"package_id"`
		CarrierCode   string  `json:"carrier_code"`
		ServiceCode   string  `json:"service_code"`
		PickupAddress Address `json:"pickup_address"`
		Destination   Address `json:"destination"`
		Parcel        Parcel  `json:"parcel"`
		ReasonCode    string  `json:"reason_code"`
	}{
		ReturnCaseID: decision.CaseID, OrderID: decision.OrderID, PackageID: decision.PackageID,
		CarrierCode: returnCarrierCode, ServiceCode: returnServiceCode,
		PickupAddress: decision.PickupAddress, Destination: decision.Destination,
		Parcel: decision.Parcel, ReasonCode: decision.ReasonCode,
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ReturnShipment{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+createReturnShipmentPath, bytes.NewReader(payload))
	if err != nil {
		return ReturnShipment{}, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", decisionIdempotencyKey(decision))

	response, err := c.http.Do(request)
	if err != nil {
		return ReturnShipment{}, fmt.Errorf("call Shipping return shipment API: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil {
		return ReturnShipment{}, fmt.Errorf("read Shipping return shipment response: %w", err)
	}
	if len(body) > 1<<20 {
		return ReturnShipment{}, errors.New("Shipping return shipment response exceeded size limit")
	}
	if response.StatusCode != http.StatusCreated {
		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
			ReturnShipment ReturnShipment `json:"return_shipment"`
		}
		_ = json.Unmarshal(body, &envelope)
		return envelope.ReturnShipment, &Error{StatusCode: response.StatusCode, Code: envelope.Error.Code}
	}
	var created ReturnShipment
	if err := json.Unmarshal(body, &created); err != nil {
		return ReturnShipment{}, fmt.Errorf("decode Shipping return shipment response: %w", err)
	}
	if created.ID == "" || created.ReturnCaseID != decision.CaseID || created.OrderID != decision.OrderID || created.PackageID != decision.PackageID || created.Status != "CREATED" || created.CODAmountVND != 0 || created.FeePayer != "SHOP" {
		return ReturnShipment{}, errors.New("Shipping response did not match the approved Care return request")
	}
	return created, nil
}

func decisionIdempotencyKey(decision ApprovedDecision) string {
	canonical := decision.CaseID + "\x00" + fmt.Sprint(decision.DecisionVersion) + "\x00" + decision.PackageID
	digest := sha256.Sum256([]byte(canonical))
	return "care-return:" + hex.EncodeToString(digest[:])
}
