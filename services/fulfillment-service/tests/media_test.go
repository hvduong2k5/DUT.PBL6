package tests

import (
	"context"
	"github.com/google/uuid"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRejectedMediaNeverHasVerifiedTimestamp(t *testing.T) {
	s, in := fixture(t)
	task := mustTask(t, s, in)
	p := principal(task)
	task = change(t, s, p, task, d.Mutation{Action: "AssignTask"})
	task = change(t, s, p, task, d.Mutation{Action: "StartTask"})
	f, e := s.CreateMedia(context.Background(), p, task.ID, "media-"+uuid.NewString(), d.MediaRequest{PackageID: task.Package.ID, Type: "video/mp4", Size: 100, Hash: d.Hash("fixture")})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CompleteMedia(context.Background(), p, f.ID, "complete-"+uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	f, e = s.File(context.Background(), f.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinishMedia(context.Background(), f, "", "", "", false); e != nil {
		t.Fatal(e)
	}
	out, e := s.File(context.Background(), f.ID)
	if e != nil || out.Status != "REJECTED" || out.VerifiedAt != nil || out.Checksum != nil {
		t.Fatal("rejected video presented as verified", e)
	}
}

func TestMediaBindingAndRequiredEvidence(t *testing.T) {
	s, in := fixture(t)
	in.Inputs.Policy.Evidence = "VIDEO_REQUIRED"
	in.Integrated = true
	task := mustTask(t, s, in)
	p := principal(task)
	task = change(t, s, p, task, d.Mutation{Action: "AssignTask"})
	task = change(t, s, p, task, d.Mutation{Action: "StartTask"})
	for _, a := range task.Allocations {
		task = change(t, s, p, task, d.Mutation{Action: "SetPickedQuantity", Child: a.ID, Counted: a.Allocated, AllocationVersion: a.Version})
	}
	for _, c := range task.Checklist {
		task = change(t, s, p, task, d.Mutation{Action: "ConfirmChecklist", Child: c.Code, Result: "PASS"})
	}
	m := d.Mutation{Action: "SealTask", Expected: task.Version, PackageID: task.Package.ID, Seal: "SEAL-MEDIA-" + strings.ToUpper(uuid.NewString()), Parcel: &d.Parcel{Weight: 1000, Length: 10, Width: 10, Height: 10, Fragile: true}}
	proof := validation(task)
	if _, e := s.Mutate(context.Background(), p, task.ID, "seal-"+uuid.NewString(), d.Hash(m), m, proof); e == nil {
		t.Fatal("required evidence bypassed")
	}
	req := d.MediaRequest{PackageID: task.Package.ID, Type: "video/mp4", Size: 1024, Hash: d.Hash("synthetic-media")}
	key := "media-" + uuid.NewString()
	file, e := s.CreateMedia(context.Background(), p, task.ID, key, req)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.CreateMedia(context.Background(), p, task.ID, key, req)
	if e != nil || again.ID != file.ID || !again.Expires.Equal(file.Expires) {
		t.Fatal("session renewed on retry", e)
	}
	queued, e := s.CompleteMedia(context.Background(), p, file.ID, "complete-"+uuid.NewString())
	if e != nil || queued.Status != "VERIFYING" {
		t.Fatal(e)
	}
	f, e := s.File(context.Background(), file.ID)
	if e != nil {
		t.Fatal(e)
	}
	// This is a metadata/state-machine fixture, NOT scanner or real S3 evidence.
	if e = s.FinishMedia(context.Background(), f, "packing-evidence/verified/"+file.ID, "fixture-version", req.Hash, true); e != nil {
		t.Fatal(e)
	}
	m.EvidenceID = &file.ID
	task, e = s.Mutate(context.Background(), p, task.ID, "seal-"+uuid.NewString(), d.Hash(m), m, validation(task))
	if e != nil {
		t.Fatal(e)
	}
	var at *time.Time
	var hold bool
	if e = pool.QueryRow(context.Background(), `SELECT attached_at,legal_hold FROM fulfillment.media_files WHERE file_id=$1`, file.ID).Scan(&at, &hold); e != nil || at == nil || !hold {
		t.Fatal("sealed evidence not protected", e)
	}
	var count int
	if e = pool.QueryRow(context.Background(), `SELECT count(*) FROM fulfillment.outbox WHERE stream_id=$1 AND topic='fulfillment.events.v2'`, task.OrderID).Scan(&count); e != nil || count != 1 {
		t.Fatal("seal handoff event not atomic", e)
	}
	var event []byte
	_ = pool.QueryRow(context.Background(), `SELECT envelope FROM fulfillment.outbox WHERE stream_id=$1 AND topic='fulfillment.events.v2'`, task.OrderID).Scan(&event)
	_ = os.MkdirAll("results", 0755)
	_ = os.WriteFile("results/packing-event.json", event, 0644)
	var audit []byte
	_ = pool.QueryRow(context.Background(), `SELECT envelope FROM fulfillment.outbox WHERE stream_id=$1 ORDER BY stream_version LIMIT 1`, "audit:"+task.OrderID).Scan(&audit)
	_ = os.WriteFile("results/audit-event.json", audit, 0644)
}
