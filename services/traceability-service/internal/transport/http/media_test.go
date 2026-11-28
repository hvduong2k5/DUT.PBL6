package httptransport

import (
	"context"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type mediaGuard struct {
	err   error
	calls int
}

func (g *mediaGuard) ValidateMedia(context.Context, domain.Content) error { g.calls++; return g.err }

func TestMediaFailurePreservesWorkflowAndPublicationReplay(t *testing.T) {
	h := scopedServer(t, true)
	h.store.MediaEnabled = true
	guard := &mediaGuard{err: domain.ErrUnavailable}
	h.workflow.Media = guard
	maker := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create, domain.Submit)
	reviewer := h.token(reviewerID, warehouseA, domain.ReadInternal, domain.Approve, domain.Publish)
	input := scopedInput(batchA, "SKU-1")
	input.Content.Evidence[0].Reference = "document:66666666-6666-4666-8666-666666666666"
	created := sendScoped(h.handler, "POST", scopedPath, encoded(input), maker, "media-create-key-0001")
	requireResponse(t, created, 201)
	p := responseProfile(t, created)
	path := scopedPath + "/" + p.ID + "/revisions/1"
	requireResponse(t, sendScoped(h.handler, "POST", path+"/submit", `{"expected_version":1}`, maker, "media-submit-key-0001"), 503)
	current, err := h.store.Get(context.Background(), p.ID)
	if err != nil || current.Version != 1 || current.Revisions[0].Status != "DRAFT" {
		t.Fatal("Media failure changed draft", err)
	}
	guard.err = nil
	requireResponse(t, sendScoped(h.handler, "POST", path+"/submit", `{"expected_version":1}`, maker, "media-submit-key-0001"), 200)
	requireResponse(t, sendScoped(h.handler, "POST", path+"/approve", `{"expected_version":2,"reason":"Independent review"}`, reviewer, "media-review-key-0001"), 200)
	guard.err = domain.ErrUnavailable
	requireResponse(t, sendScoped(h.handler, "POST", path+"/publish", `{"expected_version":3}`, reviewer, "media-publish-key-001"), 503)
	current, err = h.store.Get(context.Background(), p.ID)
	if err != nil || current.PublicRevision != nil || current.Version != 3 {
		t.Fatal("Media failure changed publication", err)
	}
	guard.err = nil
	requireResponse(t, sendScoped(h.handler, "POST", path+"/publish", `{"expected_version":3}`, reviewer, "media-publish-key-001"), 200)
	before := guard.calls
	guard.err = domain.ErrUnavailable
	requireResponse(t, sendScoped(h.handler, "POST", path+"/publish", `{"expected_version":3}`, reviewer, "media-publish-key-001"), 200)
	if guard.calls != before {
		t.Fatal("committed replay queried Media")
	}
	var jobs int
	if err := h.admin.QueryRow(context.Background(), `SELECT count(*) FROM media_bind_jobs`).Scan(&jobs); err != nil || jobs != 1 {
		t.Fatal("binding obligation was not durable and unique", jobs, err)
	}
}
