package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

type actionRequest struct {
	Action     string    `json:"action"`
	ShipmentID uuid.UUID `json:"shipment_id"`
	ReasonCode string    `json:"reason_code,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

func hashActionRequest(request actionRequest) (string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}
