package transport

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/omamx/fulfillment-service/internal/application"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"github.com/omamx/fulfillment-service/internal/persistence"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HTTP struct {
	Service *application.Service
	Media   *application.Media
	Auth    Auth
}

func response(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, r *http.Request, e error) {
	var de *d.Error
	if !errors.As(e, &de) {
		de = &d.Error{Code: "DEPENDENCY_UNAVAILABLE", HTTP: 503, Retryable: true}
	}
	if de.HTTP == 429 || de.HTTP == 503 {
		w.Header().Set("Retry-After", "5")
	}
	id := w.Header().Get("X-Request-ID")
	if id == "" {
		id = uuid.NewString()
		w.Header().Set("X-Request-ID", id)
	}
	response(w, de.HTTP, map[string]any{"code": de.Code, "message": de.Code, "request_id": id, "retryable": de.Retryable, "details": map[string]any{}})
}
func decode(r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 65536)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if e := dec.Decode(out); e != nil {
		return d.Fail("INVALID_JSON", 400)
	}
	var x any
	if dec.Decode(&x) != io.EOF {
		return d.Fail("INVALID_JSON", 400)
	}
	return nil
}
func (h HTTP) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Request-ID", uuid.NewString())
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) { response(w, 200, map[string]string{"status": "live"}) })
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		var count int
		e := h.Service.Store.DB.QueryRow(r.Context(), `SELECT count(*) FROM public.fulfillment_migrations`).Scan(&count)
		if e != nil || count < 3 {
			writeError(w, r, d.Fail("SCHEMA_UNAVAILABLE", 503))
			return
		}
		response(w, 200, map[string]any{"status": "ready", "admission_enabled": h.Service.AdmissionEnabled, "seal_enabled": h.Service.SealEnabled})
	})
	r.Route("/api/v1/admin/fulfillment", func(r chi.Router) {
		r.Use(h.Auth.Middleware)
		r.Get("/tasks", h.list)
		r.Get("/tasks/{task_id}", h.task)
		r.Get("/tasks/{task_id}/history", h.history)
		routes := []struct{ Path, Action, Method string }{
			{"assignments", "AssignTask", "POST"}, {"start", "StartTask", "POST"}, {"unassign", "UnassignTask", "POST"}, {"picks/{allocation_id}", "SetPickedQuantity", "PUT"}, {"checklist/{item_code}", "ConfirmChecklist", "PUT"}, {"blocks", "BlockTask", "POST"}, {"blocks/{block_id}/resolve", "ResolveTaskBlock", "POST"}, {"seal", "SealTask", "POST"}, {"reopen", "ReopenTask", "POST"},
		}
		for _, route := range routes {
			action := route.Action
			r.MethodFunc(route.Method, "/tasks/{task_id}/"+route.Path, func(w http.ResponseWriter, r *http.Request) { h.mutate(w, r, action) })
		}
		r.Post("/tasks/{task_id}/media-sessions", h.createMedia)
		r.Post("/media/{file_id}/complete", h.completeMedia)
		r.Get("/media/{file_id}", h.file)
		r.Post("/media/{file_id}/access", h.access)
		r.Get("/commands/{key}", h.command)
	})
	return r
}
func (h HTTP) task(w http.ResponseWriter, r *http.Request) {
	t, e := h.Service.Store.Task(r.Context(), chi.URLParam(r, "task_id"))
	if e == nil && !principal(r.Context()).Can("fulfillment.task.read", t.Warehouse) {
		e = d.Fail("FORBIDDEN", 403)
	}
	if e != nil {
		writeError(w, r, e)
		return
	}
	w.Header().Set("ETag", strconv.Quote(strconv.FormatInt(t.Version, 10)))
	response(w, 200, t)
}
func (h HTTP) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 25
	var e error
	if q.Get("limit") != "" {
		limit, e = strconv.Atoi(q.Get("limit"))
		if e != nil {
			writeError(w, r, d.Fail("INVALID_LIMIT", 400))
			return
		}
	}
	page, e := h.Service.Store.List(r.Context(), principal(r.Context()), persistence.Filter{Warehouse: q.Get("warehouse_id"), State: q.Get("workflow_state"), Assignee: q.Get("assignee_id"), Cursor: q.Get("cursor"), Limit: limit})
	if e != nil {
		writeError(w, r, e)
		return
	}
	// Queue returns the contract's summary DTO, not internal snapshots/allocations.
	b, _ := json.Marshal(page.Items)
	var items []map[string]any
	_ = json.Unmarshal(b, &items)
	allowed := []string{"task_id", "order_id", "workflow_state", "version", "ready_generation", "warehouse_id", "assignee_id", "packing_due_at", "admitted_at", "integration", "channel", "line_count", "total_quantity", "active_block_count", "priority_reason", "deadline_revision"}
	out := []map[string]any{}
	for _, t := range items {
		x := map[string]any{}
		for _, k := range allowed {
			x[k] = t[k]
		}
		out = append(out, x)
	}
	response(w, 200, map[string]any{"items": out, "next_cursor": page.Cursor, "queue_as_of": page.AsOf})
}
func (h HTTP) mutate(w http.ResponseWriter, r *http.Request, action string) {
	etag := r.Header.Get("If-Match")
	if etag == "" {
		writeError(w, r, d.Fail("IF_MATCH_REQUIRED", 428))
		return
	}
	unquoted, e := strconv.Unquote(etag)
	if e != nil {
		writeError(w, r, d.Fail("INVALID_IF_MATCH", 400))
		return
	}
	version, e := strconv.ParseInt(unquoted, 10, 64)
	if e != nil || version <= 0 || strconv.Quote(strconv.FormatInt(version, 10)) != etag {
		writeError(w, r, d.Fail("INVALID_IF_MATCH", 400))
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if !d.Key(key) {
		writeError(w, r, d.Fail("INVALID_KEY", 400))
		return
	}
	var body map[string]json.RawMessage
	if e = decode(r, &body); e != nil || body == nil {
		writeError(w, r, d.Fail("INVALID_JSON", 400))
		return
	}
	fields := map[string][]string{"AssignTask": {"assignee_id", "reason"}, "StartTask": {}, "UnassignTask": {"reason"}, "SetPickedQuantity": {"counted_quantity", "allocation_version"}, "ConfirmChecklist": {"result", "note"}, "BlockTask": {"reason_code", "reason"}, "ResolveTaskBlock": {"reason"}, "SealTask": {"package_id", "seal_code", "parcel", "primary_evidence_id"}, "ReopenTask": {"reason"}}
	for k := range body {
		ok := false
		for _, f := range fields[action] {
			if k == f {
				ok = true
			}
		}
		if !ok {
			writeError(w, r, d.Fail("UNKNOWN_FIELD", 400))
			return
		}
	}
	if action == "SetPickedQuantity" && (body["counted_quantity"] == nil || body["allocation_version"] == nil) {
		writeError(w, r, d.Fail("MISSING_PICK_INPUT", 400))
		return
	}
	raw, _ := json.Marshal(body)
	var m d.Mutation
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&m); e != nil {
		writeError(w, r, d.Fail("INVALID_JSON", 400))
		return
	}
	m.Action = action
	m.Expected = version
	m.Child = chi.URLParam(r, "allocation_id")
	if m.Child == "" {
		m.Child = chi.URLParam(r, "item_code")
	}
	if m.Child == "" {
		m.Child = chi.URLParam(r, "block_id")
	}
	// Decode to generic JSON using UseNumber so canonical integers retain precision.
	canonical := map[string]any{}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	_ = dec.Decode(&canonical)
	for _, name := range []string{"allocation_id", "item_code", "block_id"} {
		if value := chi.URLParam(r, name); value != "" {
			canonical[name] = value
		}
	}
	hash := d.Hash(map[string]any{"action": action, "task_id": chi.URLParam(r, "task_id"), "expected_version": version, "body": canonical})
	t, e := h.Service.Mutate(r.Context(), principal(r.Context()), chi.URLParam(r, "task_id"), key, hash, m)
	if e != nil {
		writeError(w, r, e)
		return
	}
	w.Header().Set("ETag", strconv.Quote(strconv.FormatInt(t.Version, 10)))
	if action == "SealTask" {
		rev, err := h.Service.Store.Revision(r.Context(), t.ID, t.Version)
		if err != nil {
			writeError(w, r, err)
			return
		}
		response(w, 200, map[string]any{"task_id": t.ID, "order_id": t.OrderID, "package_id": t.Package.ID, "workflow_state": t.State, "version": t.Version, "seal_code": t.Package.Seal, "packed_at": rev.Occurred, "integration": t.Integration})
		return
	}
	response(w, 200, t)
}
func (h HTTP) history(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	after := int64(0)
	limit := 25
	var e error
	if q.Get("cursor") != "" {
		after, e = strconv.ParseInt(q.Get("cursor"), 10, 64)
	}
	if q.Get("limit") != "" {
		limit, e = strconv.Atoi(q.Get("limit"))
	}
	if e != nil {
		writeError(w, r, d.Fail("INVALID_CURSOR", 400))
		return
	}
	items, e := h.Service.Store.History(r.Context(), principal(r.Context()), chi.URLParam(r, "task_id"), after, limit)
	if e != nil {
		writeError(w, r, e)
		return
	}
	var next *string
	if len(items) == limit {
		v := strconv.FormatInt(items[len(items)-1]["task_version"].(int64), 10)
		next = &v
	}
	response(w, 200, map[string]any{"items": items, "next_cursor": next})
}
func (h HTTP) createMedia(w http.ResponseWriter, r *http.Request) {
	var in d.MediaRequest
	if e := decode(r, &in); e != nil {
		writeError(w, r, e)
		return
	}
	if h.Media == nil {
		writeError(w, r, d.Fail("MEDIA_CAPABILITY_UNAVAILABLE", 503))
		return
	}
	out, e := h.Media.Create(r.Context(), principal(r.Context()), chi.URLParam(r, "task_id"), r.Header.Get("Idempotency-Key"), in)
	if e != nil {
		writeError(w, r, e)
		return
	}
	response(w, 201, out)
}
func (h HTTP) completeMedia(w http.ResponseWriter, r *http.Request) {
	var empty struct{}
	if e := decode(r, &empty); e != nil {
		writeError(w, r, e)
		return
	}
	if h.Media == nil || !h.Media.UploadEnabled {
		writeError(w, r, d.Fail("MEDIA_CAPABILITY_UNAVAILABLE", 503))
		return
	}
	out, e := h.Service.Store.CompleteMedia(r.Context(), principal(r.Context()), chi.URLParam(r, "file_id"), r.Header.Get("Idempotency-Key"))
	if e != nil {
		writeError(w, r, e)
		return
	}
	response(w, 202, out)
}
func (h HTTP) file(w http.ResponseWriter, r *http.Request) {
	f, e := h.Service.Store.File(r.Context(), chi.URLParam(r, "file_id"))
	if e != nil {
		writeError(w, r, e)
		return
	}
	t, e := h.Service.Store.Task(r.Context(), f.TaskID)
	if e == nil && !principal(r.Context()).Can("fulfillment.media.read_metadata", t.Warehouse) {
		e = d.Fail("FORBIDDEN", 403)
	}
	if e != nil {
		writeError(w, r, e)
		return
	}
	response(w, 200, f)
}
func (h HTTP) access(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Purpose string `json:"purpose"`
		Case    string `json:"case_reference"`
	}
	if e := decode(r, &in); e != nil {
		writeError(w, r, e)
		return
	}
	if h.Media == nil {
		writeError(w, r, d.Fail("MEDIA_CAPABILITY_UNAVAILABLE", 503))
		return
	}
	v, e := h.Media.Access(r.Context(), principal(r.Context()), chi.URLParam(r, "file_id"), in.Purpose)
	if e != nil {
		writeError(w, r, e)
		return
	}
	response(w, 200, v)
}
func (h HTTP) command(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	v, e := h.Service.Store.Command(r.Context(), principal(r.Context()), q.Get("order_id"), q.Get("action"), chi.URLParam(r, "key"))
	if e != nil {
		writeError(w, r, e)
		return
	}
	response(w, 200, v)
}
