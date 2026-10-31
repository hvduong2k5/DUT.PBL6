package simulator

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"net/http"
	"strconv"
	"time"
)

func (s *Simulator) coreLifecycleRoutes(api chi.Router) {
	api.Post("/order-core/admission-barrier", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Order      string `json:"order_id"`
			Version    int64  `json:"order_version"`
			Generation int64  `json:"generation"`
			Purpose    string `json:"purpose"`
		}
		if e := read(r, &in); e != nil || in.Version < 1 || (in.Purpose != "HOLD" && in.Purpose != "RESUME") {
			errResponse(w, domain.E("INVALID_BARRIER", 400))
			return
		}
		tx, e := s.DB.Begin(r.Context())
		if e != nil {
			errResponse(w, e)
			return
		}
		defer tx.Rollback(r.Context())
		if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "core-admission:"+in.Order); e != nil {
			errResponse(w, e)
			return
		}
		var task bool
		var expires time.Time
		if e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM core_source_history WHERE order_id=$1 AND proof->>'kind'='PACKING_ACCEPTED'),clock_timestamp()+interval '10 seconds'", in.Order).Scan(&task, &expires); e != nil {
			errResponse(w, e)
			return
		}
		if !task {
			if _, e = tx.Exec(r.Context(), "INSERT INTO core_admission_barriers(order_id,fenced_generation) VALUES($1,$2) ON CONFLICT(order_id) DO UPDATE SET fenced_generation=GREATEST(core_admission_barriers.fenced_generation,excluded.fenced_generation)", in.Order, in.Generation); e != nil {
				errResponse(w, e)
				return
			}
		}
		if e = tx.Commit(r.Context()); e != nil {
			errResponse(w, e)
			return
		}
		response(w, 200, persistence.AdmissionBarrier{Order: in.Order, Version: in.Version, Generation: in.Generation, Purpose: in.Purpose, NoTask: !task, Decision: uuid.NewString(), Expires: expires})
	})
	api.Post("/test/core-source", func(w http.ResponseWriter, r *http.Request) {
		var proof persistence.SourceEvidence
		if e := read(r, &proof); e != nil || proof.Version < 1 {
			errResponse(w, domain.E("INVALID_SOURCE", 400))
			return
		}
		if _, e := uuid.Parse(proof.Order); e != nil {
			errResponse(w, domain.E("INVALID_ORDER", 400))
			return
		}
		if _, e := uuid.Parse(proof.Resource); e != nil {
			errResponse(w, domain.E("INVALID_RESOURCE", 400))
			return
		}
		var existing []byte
		if e := s.DB.QueryRow(r.Context(), "SELECT proof FROM core_source_history WHERE source=$1 AND resource_id=$2 AND version=$3", proof.Source, proof.Resource, proof.Version).Scan(&existing); e == nil {
			var prior persistence.SourceEvidence
			if json.Unmarshal(existing, &prior) != nil {
				errResponse(w, domain.E("SOURCE_CORRUPT", 503))
				return
			}
			proof.Occurred = prior.Occurred
			if domain.Hash(proof) != domain.Hash(prior) {
				errResponse(w, domain.E("SOURCE_VERSION_CONFLICT", 409))
				return
			}
			response(w, 200, prior)
			return
		}
		if proof.Kind == "PACKING_ACCEPTED" {
			request, _ := http.NewRequestWithContext(r.Context(), "GET", s.OrderURL+"/internal/ready-authorizations/"+proof.Order, nil)
			request.Header.Set("X-Internal-Token", s.Token)
			result, e := s.Client.Do(request)
			if e != nil {
				errResponse(w, e)
				return
			}
			defer result.Body.Close()
			var ready struct {
				Authorized bool  `json:"authorized"`
				Generation int64 `json:"generation"`
			}
			if result.StatusCode != 200 || json.NewDecoder(result.Body).Decode(&ready) != nil || !ready.Authorized || ready.Generation != proof.Generation {
				errResponse(w, domain.E("READY_NOT_AUTHORIZED", 409))
				return
			}
		}
		if proof.Source == "care" {
			if proof.Kind != "CASE_OPENED" && proof.Kind != "CASE_CLOSED" {
				errResponse(w, domain.E("INVALID_SOURCE", 400))
				return
			}
			tag, e := s.DB.Exec(r.Context(), "INSERT INTO care(order_id,active) VALUES($1,$2) ON CONFLICT(order_id) DO UPDATE SET active=excluded.active WHERE NOT care.finalized", proof.Order, proof.Kind == "CASE_OPENED")
			if e != nil || tag.RowsAffected() != 1 {
				errResponse(w, domain.E("CASE_BARRIER_FINALIZED", 409))
				return
			}
		}
		if e := s.DB.QueryRow(r.Context(), "SELECT clock_timestamp()").Scan(&proof.Occurred); e != nil {
			errResponse(w, e)
			return
		}
		if e := s.recordCoreSource(r.Context(), proof); e != nil {
			errResponse(w, e)
			return
		}
		response(w, 201, proof)
	})
	api.Get("/order-core/resources/{id}", func(w http.ResponseWriter, r *http.Request) {
		version, e := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
		if e != nil || version < 1 {
			errResponse(w, domain.E("INVALID_VERSION", 400))
			return
		}
		var raw json.RawMessage
		if e = s.DB.QueryRow(r.Context(), "SELECT proof FROM core_source_history WHERE source=$1 AND resource_id=$2 AND version=$3", r.URL.Query().Get("source"), chi.URLParam(r, "id"), version).Scan(&raw); e != nil {
			errResponse(w, domain.E("RESOURCE_VERSION_UNKNOWN", 503))
			return
		}
		response(w, 200, raw)
	})
	api.Post("/order-core/completion", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Order   string `json:"order_id"`
			Version int64  `json:"order_version"`
		}
		if e := read(r, &in); e != nil || in.Version < 1 {
			errResponse(w, domain.E("INVALID_COMPLETION", 400))
			return
		}
		ctx := r.Context()
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			errResponse(w, e)
			return
		}
		defer tx.Rollback(ctx)
		if _, e = tx.Exec(ctx, "INSERT INTO care(order_id) VALUES($1) ON CONFLICT DO NOTHING", in.Order); e != nil {
			errResponse(w, e)
			return
		}
		var active bool
		if e = tx.QueryRow(ctx, "SELECT active FROM care WHERE order_id=$1 FOR UPDATE", in.Order).Scan(&active); e != nil {
			errResponse(w, e)
			return
		}
		if active {
			response(w, 200, persistence.CompletionDecision{Order: in.Order, Version: in.Version, Allowed: false})
			return
		}
		var id string
		var expires time.Time
		if e = tx.QueryRow(ctx, "INSERT INTO core_completion_decisions(order_id,order_version,decision_id,expires_at) VALUES($1,$2,$3,clock_timestamp()+interval '10 seconds') ON CONFLICT(order_id,order_version) DO UPDATE SET expires_at=excluded.expires_at RETURNING decision_id,expires_at", in.Order, in.Version, uuid.NewString()).Scan(&id, &expires); e != nil {
			errResponse(w, e)
			return
		}
		if _, e = tx.Exec(ctx, "UPDATE care SET finalized=true WHERE order_id=$1", in.Order); e != nil {
			errResponse(w, e)
			return
		}
		var settled bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM core_source_history WHERE order_id=$1 AND proof->>'kind'='SHIPMENT_DELIVERED' AND (proof->>'settlement_verified')::boolean)", in.Order).Scan(&settled); e != nil {
			errResponse(w, e)
			return
		}
		if e = tx.Commit(ctx); e != nil {
			errResponse(w, e)
			return
		}
		response(w, 200, persistence.CompletionDecision{Order: in.Order, Version: in.Version, Allowed: true, Decision: id, Expires: expires, Settlement: settled})
	})
}
func (s *Simulator) recordCoreSource(ctx context.Context, proof persistence.SourceEvidence) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if proof.Kind == "PACKING_ACCEPTED" {
		if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "core-admission:"+proof.Order); e != nil {
			return e
		}
		var fenced bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM core_admission_barriers WHERE order_id=$1 AND fenced_generation>=$2)", proof.Order, proof.Generation).Scan(&fenced); e != nil {
			return e
		}
		if fenced {
			return domain.E("ADMISSION_FENCED", 409)
		}
	}
	raw, _ := json.Marshal(proof)
	if _, e = tx.Exec(ctx, "INSERT INTO core_source_history(source,resource_id,version,order_id,proof) VALUES($1,$2,$3,$4,$5)", proof.Source, proof.Resource, proof.Version, proof.Order, raw); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
