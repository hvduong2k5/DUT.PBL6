package simulator

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/omamx/order-service/internal/domain"
)

// Catalogue maintenance is the persistent barrier used while the
// administrative catalogue changes (address-seam design, PR-B). The flag
// lives in the simulator database (bootstrap schema in schema.sql,
// applied at every simulator start) so it survives restarts,
// redeploys and rollbacks; nothing clears it automatically. While it is on,
// the checkout provider paths (address validate, terms/quote, revalidate)
// answer 503 CATALOGUE_MAINTENANCE. The check fails closed: until the flag
// has been read successfully, checkout is treated as under maintenance.
//
// A caller that presents the internal token in the X-Catalogue-Bypass
// header (the controlled internal smoke path of the catalogue-change
// runbook) is let through; the order core never sends that header, so
// storefront traffic cannot bypass the barrier.
const catalogueBypassHeader = "X-Catalogue-Bypass"

var catalogueMaintenanceCache struct {
	sync.Mutex
	at time.Time
	on bool
}

func (s *Simulator) catalogueMaintenanceOn(ctx context.Context) bool {
	now := time.Now()
	catalogueMaintenanceCache.Lock()
	defer catalogueMaintenanceCache.Unlock()
	if now.Sub(catalogueMaintenanceCache.at) < 2*time.Second {
		return catalogueMaintenanceCache.on
	}
	on := true // fail closed until the flag has been read successfully
	if s.DB != nil {
		var enabled bool
		if e := s.DB.QueryRow(ctx, "SELECT enabled FROM catalogue_maintenance WHERE singleton").Scan(&enabled); e == nil {
			on = enabled
		}
	}
	catalogueMaintenanceCache.on, catalogueMaintenanceCache.at = on, now
	return on
}

// catalogueBlocked writes the 503 maintenance error and reports true when
// this request must not proceed.
func (s *Simulator) catalogueBlocked(w http.ResponseWriter, r *http.Request) bool {
	if s.Token != "" && r.Header.Get(catalogueBypassHeader) == s.Token {
		return false
	}
	if !s.catalogueMaintenanceOn(r.Context()) {
		return false
	}
	errResponse(w, domain.E("CATALOGUE_MAINTENANCE", 503))
	return true
}

func (s *Simulator) setCatalogueMaintenance(ctx context.Context, enabled bool, reason string) error {
	_, e := s.DB.Exec(ctx, "INSERT INTO catalogue_maintenance(singleton, enabled, reason, updated_at) VALUES(TRUE, $1, $2, clock_timestamp()) ON CONFLICT (singleton) DO UPDATE SET enabled = EXCLUDED.enabled, reason = EXCLUDED.reason, updated_at = EXCLUDED.updated_at", enabled, reason)
	if e != nil {
		return e
	}
	catalogueMaintenanceCache.Lock()
	catalogueMaintenanceCache.on, catalogueMaintenanceCache.at = enabled, time.Now()
	catalogueMaintenanceCache.Unlock()
	return nil
}
