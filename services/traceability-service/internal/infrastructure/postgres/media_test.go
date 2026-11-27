package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func TestMediaPublicationQueueIsAtomicAndRetryable(t *testing.T) {
	ctx, store, p, snapshot := reviewSetup(t)
	store.MediaEnabled = true
	content := p.Revisions[0].Content
	file := "77777777-7777-4777-8777-777777777777"
	content.Evidence[0].Reference = "document:" + file
	meta := draftMetadata()
	meta.Key = "media-edit"
	if _, err := store.EditDraft(ctx, meta, p.ID, 1, domain.EditDraftRequest{ExpectedVersion: 1, Content: content}); err != nil {
		t.Fatal(err)
	}
	meta.Key = "media-submit"
	if _, err := store.SubmitRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 2}, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideRevision(ctx, reviewMeta("media-approve"), p.ID, 1, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 3, Reason: "Independent media review"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(ctx, `CREATE FUNCTION fail_media_job() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic queue failure'; END $$;
 CREATE TRIGGER fail_media_job BEFORE INSERT ON media_bind_jobs FOR EACH ROW EXECUTE FUNCTION fail_media_job()`); err != nil {
		t.Fatal(err)
	}
	command := domain.RevisionCommand{ExpectedVersion: 4}
	if _, err := store.PublishRevision(ctx, reviewMeta("media-publish"), p.ID, 1, command, snapshot); err == nil {
		t.Fatal("publication committed without binding obligation")
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.PublicRevision != nil || current.Version != 4 {
		t.Fatal("failed publish changed public state", err)
	}
	if _, err := store.DB.Exec(ctx, `DROP TRIGGER fail_media_job ON media_bind_jobs`); err != nil {
		t.Fatal(err)
	}
	result, err := store.PublishRevision(ctx, reviewMeta("media-publish"), p.ID, 1, command, snapshot)
	if err != nil || result.Replayed {
		t.Fatal(err)
	}
	replay, err := store.PublishRevision(ctx, reviewMeta("media-publish"), p.ID, 1, command, snapshot)
	if err != nil || !replay.Replayed {
		t.Fatal(err)
	}
	var count int
	if err := store.DB.QueryRow(ctx, `SELECT count(*) FROM media_bind_jobs`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate binding obligations", count, err)
	}
	found, err := store.BindMediaJob(ctx, func(context.Context, string, string) error { return errors.New("simulated peer outage") })
	if err != nil || !found {
		t.Fatal(found, err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE media_bind_jobs SET retry_at=now()`); err != nil {
		t.Fatal(err)
	}
	calls := 0
	found, err = store.BindMediaJob(ctx, func(_ context.Context, profile, fileID string) error {
		calls++
		if profile != p.ID || fileID != file {
			t.Error("incorrect binding identity")
		}
		// A concurrent worker cannot claim the same live lease.
		duplicate, err := store.BindMediaJob(ctx, func(context.Context, string, string) error { t.Error("duplicate lease"); return nil })
		if err != nil || duplicate {
			t.Error("binding lease not exclusive", err)
		}
		return nil
	})
	if err != nil || !found || calls != 1 {
		t.Fatal(found, calls, err)
	}
	var completed bool
	var attempts int
	if err := store.DB.QueryRow(ctx, `SELECT bound_at IS NOT NULL,attempts FROM media_bind_jobs`).Scan(&completed, &attempts); err != nil || !completed || attempts != 2 {
		t.Fatal(completed, attempts, err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE media_bind_jobs SET bound_at=NULL`); err == nil {
		t.Fatal("completed obligation was reset")
	}
	found, err = store.BindMediaJob(ctx, func(context.Context, string, string) error { t.Fatal("completed job replayed"); return nil })
	if err != nil || found {
		t.Fatal(found, err)
	}
}
