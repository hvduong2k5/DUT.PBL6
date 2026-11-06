package postgres

import (
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
)

func TestProfilePagination(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store, _ := NewProfiles(pool, "sandbox")
	empty, err := store.List(ctx, domain.ListQuery{})
	if err != nil || len(empty.Items) != 0 || empty.NextCursor != nil {
		t.Fatal("invalid empty page")
	}
	for _, batch := range []string{"batch-a", "batch-b", "batch-c"} {
		request := createRequest()
		request.BatchID = batch
		meta := draftMetadata()
		meta.Key = batch
		if _, err = store.CreateDraft(ctx, meta, request); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.List(ctx, domain.ListQuery{PageSize: 2})
	if err != nil || len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatal("invalid first page", err)
	}
	second, err := store.List(ctx, domain.ListQuery{PageSize: 2, Cursor: *first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.NextCursor != nil {
		t.Fatal("invalid second page", err)
	}
	seen := map[string]bool{}
	for _, item := range append(first.Items, second.Items...) {
		if seen[item.ID] {
			t.Fatal("duplicate across pages")
		}
		seen[item.ID] = true
	}
	for _, query := range []domain.ListQuery{{PageSize: -1}, {PageSize: 101}, {Cursor: "invalid"}} {
		if _, err = store.List(ctx, query); err == nil {
			t.Fatal("invalid pagination accepted")
		}
	}
}
