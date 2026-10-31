package httptransport

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func TestSignedEndToEndPublishQRAndSuspension(t *testing.T) {
	h, _, store, token := workflowServer(t)
	editor := token(editorID, domain.ReadInternal, domain.Create, domain.Edit, domain.Submit)
	reviewer := token(reviewerID, domain.ReadInternal, domain.Approve, domain.Publish, domain.Suspend, domain.Archive)
	created := sendDraft(h, "POST", recordsPath, encoded(completeDraftInput()), editor, "create")
	if created.Code != 201 {
		t.Fatal(created.Code)
	}
	var p domain.Profile
	_ = json.Unmarshal(created.Body.Bytes(), &p)
	root := recordsPath + "/" + p.ID
	revision := root + "/revisions/1"
	publicPath := "/api/v1/trace/" + p.QRCode
	if before := sendDraft(h, "GET", publicPath, "", "", ""); before.Code != 404 {
		t.Fatal("Draft visible")
	}
	for _, step := range []struct{ route, body, actor, key string }{{"submit", `{"expected_version":1}`, editor, "submit"}, {"approve", `{"expected_version":2,"reason":"Independent proof checked"}`, reviewer, "approve"}, {"publish", `{"expected_version":3}`, reviewer, "publish"}} {
		response := sendDraft(h, "POST", revision+"/"+step.route, step.body, step.actor, step.key)
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", step.route, response.Code, response.Body.String())
		}
		if step.route != "publish" {
			if before := sendDraft(h, "GET", publicPath, "", "", ""); before.Code != 404 {
				t.Fatal("submit/approve implicitly public")
			}
		} else {
			var trace string
			if err := store.DB.QueryRow(context.Background(), `SELECT payload->>'traceparent' FROM outbox_events WHERE aggregate_version=4`).Scan(&trace); err != nil || trace != response.Header().Get("Traceparent") || !strings.HasPrefix(trace, "00-4bf92f3577b34da6a3ce929d0e0e4736-") || !strings.HasSuffix(trace, "-00") {
				t.Fatal("request trace was not propagated to the durable event")
			}
		}
	}
	public := sendDraft(h, "GET", publicPath, "", "", "")
	if public.Code != 200 {
		t.Fatalf("public: %d %s", public.Code, public.Body.String())
	}
	var record domain.PublicRecord
	if err := json.Unmarshal(public.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Revision != 1 || record.Mode != "sandbox" || record.VerificationMethod != "MANUAL" || record.FoodSafetyCert != nil || record.ArtisanName != nil || record.CraftingVideoURL != nil {
		t.Fatal("public metadata/nullability fabricated")
	}
	for _, secret := range []string{"internal/synthetic-proof", "contributors", "created_by", "actor_id", "evidence_ids", "scope_id"} {
		if strings.Contains(public.Body.String(), secret) {
			t.Fatalf("public leak %s", secret)
		}
	}
	cloned := sendDraft(h, "POST", root+"/revisions", `{"expected_version":4,"from_revision":1}`, editor, "new-revision")
	if cloned.Code != 201 {
		t.Fatalf("clone: %d %s", cloned.Code, cloned.Body.String())
	}
	updated := completeDraftInput().Content
	updated.ProductName = "Second revision name"
	edit := sendDraft(h, "PATCH", root+"/revisions/2", encoded(domain.EditDraftRequest{ExpectedVersion: 5, Content: updated}), editor, "edit-2")
	if edit.Code != 200 {
		t.Fatal(edit.Code)
	}
	before := sendDraft(h, "GET", publicPath, "", "", "")
	if before.Code != 200 || before.Body.String() != public.Body.String() {
		t.Fatal("Draft edit changed public snapshot")
	}
	for _, step := range []struct{ route, body, actor, key string }{{"submit", `{"expected_version":6}`, editor, "submit-2"}, {"approve", `{"expected_version":7,"reason":"Second revision reviewed"}`, reviewer, "approve-2"}, {"publish", `{"expected_version":8}`, reviewer, "publish-2"}} {
		response := sendDraft(h, "POST", root+"/revisions/2/"+step.route, step.body, step.actor, step.key)
		if response.Code != 200 {
			t.Fatalf("second %s: %d %s", step.route, response.Code, response.Body.String())
		}
	}
	current := sendDraft(h, "GET", publicPath, "", "", "")
	if current.Code != 200 || !strings.Contains(current.Body.String(), updated.ProductName) {
		t.Fatal("QR did not point to new publication")
	}
	if missing := sendDraft(h, "POST", root+"/suspend", `{"expected_version":9}`, reviewer, "missing-reason"); missing.Code != 400 {
		t.Fatal("suspend without reason")
	}
	suspended := sendDraft(h, "POST", root+"/suspend", `{"expected_version":9,"reason":"Internal operator note"}`, reviewer, "suspend")
	if suspended.Code != 200 {
		t.Fatal(suspended.Code, suspended.Body.String())
	}
	gone := sendDraft(h, "GET", publicPath, "", "", "")
	if gone.Code != 410 || !strings.Contains(gone.Body.String(), "PROFILE_SUSPENDED") || strings.Contains(gone.Body.String(), updated.ProductName) || strings.Contains(gone.Body.String(), "Internal operator note") {
		t.Fatal("suspended content leaked")
	}
	if bypass := sendDraft(h, "POST", root+"/revisions", `{"expected_version":10,"from_revision":2}`, editor, "bypass"); bypass.Code != 409 {
		t.Fatal("clone bypassed suspension")
	}
	if bypass := sendDraft(h, "POST", recordsPath, encoded(completeDraftInput()), editor, "duplicate-batch"); bypass.Code != 409 {
		t.Fatal("new profile bypassed same Batch suspension")
	}
	archived := sendDraft(h, "POST", root+"/archive", `{"expected_version":10,"reason":"Preserve history"}`, reviewer, "archive")
	if archived.Code != 200 {
		t.Fatal(archived.Code)
	}
	gone = sendDraft(h, "GET", publicPath, "", "", "")
	if gone.Code != 410 || !strings.Contains(gone.Body.String(), "PROFILE_ARCHIVED") {
		t.Fatal("archive not reflected")
	}
	for _, response := range []*httptest.ResponseRecorder{public, before, current, gone} {
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cache-control missing")
		}
	}
	var events int
	if err := store.DB.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE status='PENDING'`).Scan(&events); err != nil || events != 4 {
		t.Fatalf("durable events: %d %v", events, err)
	}
}

func TestPublicHTTPReviewDueAndBatchExpiryWithoutWorker(t *testing.T) {
	for _, scenario := range []string{"review_due", "batch_expiry"} {
		t.Run(scenario, func(t *testing.T) {
			h, workflow, store, token := workflowServer(t)
			editor := token(editorID, domain.ReadInternal, domain.Create, domain.Submit)
			reviewer := token(reviewerID, domain.ReadInternal, domain.Approve, domain.Publish)
			request := completeDraftInput()
			deadline := time.Date(2026, 10, 17, 13, 0, 0, 0, time.UTC)
			reason := "BATCH_EXPIRED"
			if scenario == "review_due" {
				request.Content.Certificates[0].ValidUntil = nil
				request.Content.Certificates[0].NoExpiryStated = true
				request.Content.Certificates[0].ReviewDueAt = &deadline
				reason = "CERTIFICATE_REVIEW_DUE"
			} else {
				original := workflow.Dependencies
				workflow.Dependencies = dependencyFunc(func(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
					s, err := original.Snapshot(ctx, b)
					s.BestBefore = deadline
					s.ExpiryDate = "2026-10-17"
					return s, err
				})
			}
			created := sendDraft(h, "POST", recordsPath, encoded(request), editor, "create")
			if created.Code != 201 {
				t.Fatal(created.Code, created.Body.String())
			}
			var profile domain.Profile
			_ = json.Unmarshal(created.Body.Bytes(), &profile)
			revision := recordsPath + "/" + profile.ID + "/revisions/1"
			for _, step := range []struct{ path, body, actor string }{{"submit", `{"expected_version":1}`, editor}, {"approve", `{"expected_version":2,"reason":"Checked"}`, reviewer}, {"publish", `{"expected_version":3}`, reviewer}} {
				if response := sendDraft(h, "POST", revision+"/"+step.path, step.body, step.actor, step.path); response.Code != 200 {
					t.Fatal(step.path, response.Code, response.Body.String())
				}
			}
			store.Clock = func() time.Time { return deadline.Add(-time.Nanosecond) }
			public := sendDraft(h, "GET", "/api/v1/trace/"+profile.QRCode, "", "", "")
			if public.Code != 200 {
				t.Fatal("early cutoff", public.Code)
			}
			store.Clock = func() time.Time { return deadline }
			public = sendDraft(h, "GET", "/api/v1/trace/"+profile.QRCode, "", "", "")
			if public.Code != 410 || !strings.Contains(public.Body.String(), reason) || strings.Contains(public.Body.String(), "product_name") {
				t.Fatal("read-time cutoff failed", public.Code, public.Body.String())
			}
		})
	}
}

func TestPublishRechecksSourceAndCanReplayWhileUnavailable(t *testing.T) {
	h, workflow, store, token := workflowServer(t)
	editor := token(editorID, domain.ReadInternal, domain.Create, domain.Submit)
	reviewer := token(reviewerID, domain.ReadInternal, domain.Approve, domain.Publish)
	created := sendDraft(h, "POST", recordsPath, encoded(completeDraftInput()), editor, "create")
	if created.Code != 201 {
		t.Fatal(created.Code)
	}
	var p domain.Profile
	_ = json.Unmarshal(created.Body.Bytes(), &p)
	revision := recordsPath + "/" + p.ID + "/revisions/1"
	if response := sendDraft(h, "POST", revision+"/submit", `{"expected_version":1}`, editor, "submit"); response.Code != 200 {
		t.Fatal(response.Code)
	}
	if response := sendDraft(h, "POST", revision+"/publish", `{"expected_version":2}`, reviewer, "premature-publish"); response.Code != 409 {
		t.Fatal("published without approval")
	}
	if response := sendDraft(h, "POST", revision+"/approve", `{"expected_version":2,"reason":"Checked"}`, reviewer, "approve"); response.Code != 200 {
		t.Fatal(response.Code)
	}
	original := workflow.Dependencies
	workflow.Dependencies = dependencyFunc(func(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
		s, err := original.Snapshot(ctx, b)
		s.BatchVersion = "corrected-2"
		return s, err
	})
	if response := sendDraft(h, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "publish"); response.Code != 409 {
		t.Fatal("source correction accepted", response.Body.String())
	}
	current, err := store.Get(context.Background(), p.ID)
	if err != nil || current.Version != 3 || current.PublicRevision != nil {
		t.Fatal("failed recheck changed profile")
	}
	workflow.Dependencies = dependencyFunc(func(context.Context, domain.Binding) (domain.DependencySnapshot, error) {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	})
	if response := sendDraft(h, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "publish"); response.Code != 503 {
		t.Fatal("unavailable provider fell back")
	}
	workflow.Dependencies = original
	published := sendDraft(h, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "publish")
	if published.Code != 200 {
		t.Fatal(published.Code, published.Body.String())
	}
	workflow.Dependencies = dependencyFunc(func(context.Context, domain.Binding) (domain.DependencySnapshot, error) {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	})
	replay := sendDraft(h, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "publish")
	if replay.Code != 200 || replay.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Fatal("committed publish replay called provider")
	}
	public := sendDraft(h, "GET", "/api/v1/trace/"+p.QRCode, "", "", "")
	if public.Code != 200 {
		t.Fatal("public read synchronously depended on provider")
	}
}
