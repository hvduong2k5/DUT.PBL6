package transport

import (
	"net/http"
	"strconv"

	"github.com/omamx/order-service/internal/domain"
)

// Minimal authoritative projection; carrier timeline remains a separate provider capability.
type orderTracking struct {
	ShipmentID  *string `json:"shipment_id"`
	OrderStatus string  `json:"order_status"`
}

func ownedOrderLimit(r *http.Request) (int, error) {
	limit := 20
	for name, values := range r.URL.Query() {
		if len(values) != 1 {
			return 0, domain.E("INVALID_QUERY", 400)
		}
		switch name {
		case "limit":
			n, err := strconv.Atoi(values[0])
			if err != nil || n < 1 || n > 100 {
				return 0, domain.E("INVALID_LIMIT", 400)
			}
			limit = n
		case "cursor":
			if len(values[0]) > 4096 {
				return 0, domain.E("INVALID_CURSOR", 400)
			}
		case "status":
			if len(values[0]) > 64 {
				return 0, domain.E("INVALID_QUERY", 400)
			}
		default:
			return 0, domain.E("UNSUPPORTED_QUERY", 400)
		}
	}
	return limit, nil
}
