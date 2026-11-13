package httptransport

import (
	"net/http"
	"strconv"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func registerWorkflow(mux *http.ServeMux, cfg HandlerConfig) {
	mux.HandleFunc("GET /api/v1/trace/{qr_code}", func(w http.ResponseWriter, r *http.Request) {
		record, err := cfg.Workflow.Public(r.Context(), r.PathValue("qr_code"))
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, record)
	})
	mux.HandleFunc("POST /api/v1/admin/trace/records/{id}/revisions", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok || !permitted(w, p, domain.Edit) {
			return
		}
		if !domain.ValidUUID(r.PathValue("id")) {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		var request domain.NewRevisionRequest
		if !decodeRequest(w, r, &request, "expected_version", "from_revision") {
			return
		}
		result, err := cfg.Workflow.NewRevision(r.Context(), p, metadata(w, r), r.PathValue("id"), request)
		respondCommand(w, result, err)
	})
	for _, spec := range []struct{ route, action, permission string }{{"submit", domain.ActionSubmit, domain.Submit}, {"approve", domain.ActionApprove, domain.Approve}, {"reject", domain.ActionReject, domain.Approve}, {"publish", domain.ActionPublish, domain.Publish}} {
		mux.HandleFunc("POST /api/v1/admin/trace/records/{id}/revisions/{revision}/"+spec.route, func(w http.ResponseWriter, r *http.Request) {
			p, ok := authenticate(w, r, cfg.Identity)
			if !ok || !permitted(w, p, spec.permission) {
				return
			}
			revision, err := strconv.Atoi(r.PathValue("revision"))
			if err != nil || revision < 1 || !domain.ValidUUID(r.PathValue("id")) {
				respondError(w, domain.ErrInvalidCommand)
				return
			}
			var request domain.RevisionCommand
			required := []string{"expected_version"}
			if spec.action == domain.ActionApprove || spec.action == domain.ActionReject {
				required = append(required, "reason")
			}
			if !decodeRequest(w, r, &request, required...) {
				return
			}
			var result domain.CommandResult
			if spec.action == domain.ActionSubmit {
				result, err = cfg.Workflow.Submit(r.Context(), p, metadata(w, r), r.PathValue("id"), revision, request)
			} else if spec.action == domain.ActionPublish {
				result, err = cfg.Workflow.Publish(r.Context(), p, metadata(w, r), r.PathValue("id"), revision, request)
			} else {
				result, err = cfg.Workflow.Decide(r.Context(), p, metadata(w, r), r.PathValue("id"), revision, spec.action, request)
			}
			respondCommand(w, result, err)
		})
	}
	for _, spec := range []struct{ route, status, permission string }{{"suspend", "SUSPENDED", domain.Suspend}, {"archive", "ARCHIVED", domain.Archive}} {
		mux.HandleFunc("POST /api/v1/admin/trace/records/{id}/"+spec.route, func(w http.ResponseWriter, r *http.Request) {
			p, ok := authenticate(w, r, cfg.Identity)
			if !ok || !permitted(w, p, spec.permission) {
				return
			}
			if !domain.ValidUUID(r.PathValue("id")) {
				respondError(w, domain.ErrInvalidCommand)
				return
			}
			var request domain.RevisionCommand
			if !decodeRequest(w, r, &request, "expected_version", "reason") {
				return
			}
			result, err := cfg.Workflow.SetStatus(r.Context(), p, metadata(w, r), r.PathValue("id"), spec.status, request)
			respondCommand(w, result, err)
		})
	}
}
