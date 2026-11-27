package httptransport

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

var v2KeyPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{16,128}$`)

type v2Writer struct{ http.ResponseWriter }

func (v2Writer) v2Envelope()          {}
func isV2(w http.ResponseWriter) bool { _, ok := w.(interface{ v2Envelope() }); return ok }

func onlyQueries(r *http.Request, allowed ...string) bool {
	for key, values := range r.URL.Query() {
		found := false
		for _, name := range allowed {
			if name == key {
				found = true
			}
		}
		if !found || len(values) != 1 {
			return false
		}
	}
	return true
}

func scopedProfile(profile domain.Profile) map[string]json.RawMessage {
	data, _ := json.Marshal(profile)
	var result map[string]json.RawMessage
	_ = json.Unmarshal(data, &result)
	result["warehouse_id"], _ = json.Marshal(profile.WarehouseID)
	return result
}
func respondScopedCommand(w http.ResponseWriter, result domain.CommandResult, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	var profile domain.Profile
	if json.Unmarshal(result.Body, &profile) != nil {
		respondError(w, domain.ErrUnavailable)
		return
	}
	if result.Replayed {
		w.Header().Set("X-Idempotency-Replayed", "true")
	}
	writeJSON(w, result.Status, scopedProfile(profile))
}

func registerScoped(mux *http.ServeMux, cfg HandlerConfig) {
	if cfg.Scoped == nil {
		return
	}
	const base = "/api/v2/admin/trace"
	mux.HandleFunc("GET "+base+"/records", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !onlyQueries(r, "page_size", "cursor") {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		query := domain.ListQuery{Cursor: r.URL.Query().Get("cursor")}
		if raw, exists := r.URL.Query()["page_size"]; exists {
			n, err := strconv.Atoi(raw[0])
			if err != nil || n < 1 || n > 100 {
				respondError(w, domain.ErrInvalidCommand)
				return
			}
			query.PageSize = n
		}
		if len(query.Cursor) > 256 {
			respondError(w, domain.ErrInvalidCursor)
			return
		}
		page, err := cfg.Scoped.List(r.Context(), p, query)
		if err != nil {
			respondError(w, err)
			return
		}
		items := make([]map[string]json.RawMessage, 0, len(page.Items))
		for _, profile := range page.Items {
			items = append(items, scopedProfile(profile))
		}
		writeJSON(w, 200, map[string]any{"items": items, "total": len(items), "next_cursor": page.NextCursor})
	})
	mux.HandleFunc("GET "+base+"/batches/{batch_id}/context", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !onlyQueries(r, "sku_code") {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		result, err := cfg.Scoped.Context(r.Context(), p, r.PathValue("batch_id"), r.URL.Query().Get("sku_code"))
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, result)
	})
	mux.HandleFunc("GET "+base+"/records/by-batch", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !onlyQueries(r, "product_id", "sku_code", "source_namespace", "batch_id") {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		q := r.URL.Query()
		binding := domain.Binding{ProductID: q.Get("product_id"), SKUCode: q.Get("sku_code"), SourceNamespace: q.Get("source_namespace"), BatchID: q.Get("batch_id")}
		profiles, err := cfg.Scoped.Lookup(r.Context(), p, binding)
		if err != nil {
			respondError(w, err)
			return
		}
		items := []map[string]any{}
		for _, profile := range profiles {
			items = append(items, map[string]any{"profile_id": profile.ID, "version": profile.Version})
		}
		writeJSON(w, 200, map[string]any{"items": items, "total": len(items)})
	})
	mux.HandleFunc("GET "+base+"/records/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !onlyQueries(r) {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		profile, err := cfg.Scoped.Get(r.Context(), p, r.PathValue("id"))
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, scopedProfile(profile))
	})
	mux.HandleFunc("POST "+base+"/records", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		var request domain.CreateDraftRequest
		if !decodeRequest(w, r, &request, "product_id", "sku_code", "source_namespace", "batch_id", "content") {
			return
		}
		result, err := cfg.Scoped.Create(r.Context(), p, metadata(w, r), request)
		respondScopedCommand(w, result, err)
	})
	mux.HandleFunc("PATCH "+base+"/records/{id}/revisions/{revision}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		number, err := strconv.Atoi(r.PathValue("revision"))
		if err != nil || number < 1 {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		var request domain.EditDraftRequest
		if !decodeRequest(w, r, &request, "expected_version", "content") {
			return
		}
		result, err := cfg.Scoped.Edit(r.Context(), p, metadata(w, r), r.PathValue("id"), number, request)
		respondScopedCommand(w, result, err)
	})
	mux.HandleFunc("POST "+base+"/records/{id}/revisions", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		var request domain.NewRevisionRequest
		if !decodeRequest(w, r, &request, "expected_version", "from_revision") {
			return
		}
		result, err := cfg.Scoped.NewRevision(r.Context(), p, metadata(w, r), r.PathValue("id"), request)
		respondScopedCommand(w, result, err)
	})
	for _, action := range []struct {
		path, name       string
		revision, reason bool
	}{
		{"revisions/{revision}/submit", domain.ActionSubmit, true, false}, {"revisions/{revision}/approve", domain.ActionApprove, true, true},
		{"revisions/{revision}/reject", domain.ActionReject, true, true}, {"revisions/{revision}/publish", domain.ActionPublish, true, false},
		{"suspend", "profile.suspend", false, true}, {"archive", "profile.archive", false, true},
	} {
		mux.HandleFunc("POST "+base+"/records/{id}/"+action.path, func(w http.ResponseWriter, r *http.Request) {
			p, ok := authenticate(w, r, cfg.Identity)
			if !ok {
				return
			}
			number := 0
			if action.revision {
				var err error
				number, err = strconv.Atoi(r.PathValue("revision"))
				if err != nil || number < 1 {
					respondError(w, domain.ErrInvalidCommand)
					return
				}
			}
			var request domain.RevisionCommand
			required := []string{"expected_version"}
			if action.reason {
				required = append(required, "reason")
			}
			if !decodeRequest(w, r, &request, required...) {
				return
			}
			result, err := cfg.Scoped.Command(r.Context(), p, metadata(w, r), r.PathValue("id"), number, action.name, request)
			respondScopedCommand(w, result, err)
		})
	}
}
