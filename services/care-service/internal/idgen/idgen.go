package idgen

import (
	"strings"

	"github.com/google/uuid"
)

func New() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

func TicketCode(id uuid.UUID) string {
	compact := strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))
	return "TKT-" + compact[:12]
}
