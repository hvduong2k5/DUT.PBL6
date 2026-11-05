package usecase_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/tests/testhelper"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEditorialLifecyclePreservesPublishedSnapshot(t *testing.T) {
	ctx := context.Background()
	s := usecase.New(testhelper.NewMemory())
	editor := testhelper.Actor("CONTENT_EDITOR")
	approver := testhelper.Actor("CONTENT_APPROVER")
	a, err := s.Create(ctx, testhelper.Content("hue-story"), editor)
	require.NoError(t, err)
	require.EqualValues(t, 1, a.Version)
	_, err = s.GetPublished(ctx, "hue-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = s.Transition(ctx, a.ID, a.Version, "publish", "", editor)
	require.ErrorIs(t, err, domain.ErrForbidden)
	a, err = s.Transition(ctx, a.ID, a.Version, "submit", "", editor)
	require.NoError(t, err)
	_, err = s.Transition(ctx, a.ID, a.Version, "approve", "", editor)
	require.ErrorIs(t, err, domain.ErrForbidden)
	body := "new"
	_, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{Body: &body}, editor)
	require.ErrorIs(t, err, domain.ErrState)
	_, err = s.Transition(ctx, a.ID, a.Version, "reject", "  ", approver)
	require.ErrorIs(t, err, domain.ErrValidation)
	a, err = s.Transition(ctx, a.ID, a.Version, "reject", "Need sources", approver)
	require.NoError(t, err)
	require.Equal(t, "Need sources", a.Decisions[0].Reason)
	body = "Reviewed story"
	a, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{Body: &body}, editor)
	require.NoError(t, err)
	require.Len(t, a.Revisions, 2)
	for _, action := range []string{"submit", "approve", "publish"} {
		a, err = s.Transition(ctx, a.ID, a.Version, action, "", approver)
		require.NoError(t, err)
	}
	pub, err := s.GetPublished(ctx, "hue-story")
	require.NoError(t, err)
	publishedID := *pub.PublishedRevisionID
	body = "Unapproved secret"
	slug := "hue-story-v2"
	a, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{Body: &body, Slug: &slug}, editor)
	require.NoError(t, err)
	require.Equal(t, domain.Draft, a.Current().Status)
	pub, err = s.GetPublished(ctx, "hue-story")
	require.NoError(t, err)
	require.Equal(t, "Reviewed story", pub.Revision(publishedID).Content.Body)
	require.Equal(t, domain.Published, a.Revision(publishedID).Status)
	_, err = s.GetPublished(ctx, slug)
	require.ErrorIs(t, err, domain.ErrNotFound)
	a, err = s.Transition(ctx, a.ID, a.Version, "hide", "", editor)
	require.NoError(t, err)
	require.False(t, a.Visible)
	_, err = s.GetPublished(ctx, "hue-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	for _, action := range []string{"submit", "approve", "publish"} {
		a, err = s.Transition(ctx, a.ID, a.Version, action, "", approver)
		require.NoError(t, err)
	}
	_, err = s.GetPublished(ctx, "hue-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	pub, err = s.GetPublished(ctx, slug)
	require.NoError(t, err)
	require.Equal(t, "Unapproved secret", pub.Revision(*pub.PublishedRevisionID).Content.Body)
	require.Len(t, a.Slugs, 2)
	require.Equal(t, "Reviewed story", a.Revision(publishedID).Content.Body)
}
func TestPublishDeniedBeforeRepositoryAccess(t *testing.T) {
	for _, tc := range []struct {
		name  string
		actor domain.Identity
		want  error
	}{
		{"editor", testhelper.Actor("CONTENT_EDITOR"), domain.ErrForbidden},
		{"other role", testhelper.Actor("CUSTOMER"), domain.ErrForbidden},
		{"missing identity", domain.Identity{}, domain.ErrUnauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := usecase.New(nil)
			a, err := s.Transition(context.Background(), uuid.New(), 1, "publish", "", tc.actor)
			require.ErrorIs(t, err, tc.want)
			require.Nil(t, a)
		})
	}
}

func TestCreateUpdateConflictsAndValidation(t *testing.T) {
	ctx := context.Background()
	s := usecase.New(testhelper.NewMemory())
	actor := testhelper.Actor("ADMIN")
	a, err := s.Create(ctx, testhelper.Content("unique"), actor)
	require.NoError(t, err)
	_, err = s.Create(ctx, testhelper.Content("unique"), actor)
	require.ErrorIs(t, err, domain.ErrSlug)
	_, err = s.Get(ctx, uuid.New(), actor)
	require.ErrorIs(t, err, domain.ErrNotFound)
	title := "Updated"
	a, err = s.Update(ctx, a.ID, 1, usecase.Patch{Title: &title}, actor)
	require.NoError(t, err)
	require.Equal(t, title, a.Current().Content.Title)
	_, err = s.Update(ctx, a.ID, 1, usecase.Patch{Title: &title}, actor)
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{}, actor)
	require.ErrorIs(t, err, domain.ErrValidation)
	_, err = s.Update(ctx, a.ID, 0, usecase.Patch{Title: &title}, actor)
	require.ErrorIs(t, err, domain.ErrValidation)
	_, err = s.Create(ctx, testhelper.Content("bad slug"), actor)
	require.ErrorIs(t, err, domain.ErrValidation)
	_, err = s.Create(ctx, testhelper.Content("other"), domain.Identity{})
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
	_, err = s.Create(ctx, testhelper.Content("other"), testhelper.Actor("CUSTOMER"))
	require.ErrorIs(t, err, domain.ErrForbidden)
	items, total, err := s.List(ctx, domain.Filter{Page: 1, Limit: 10, Status: domain.Draft, Query: "updated"}, actor)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	_, _, err = s.List(ctx, domain.Filter{Page: 1, Limit: 101, Public: true}, domain.Identity{})
	require.ErrorIs(t, err, domain.ErrValidation)
}
