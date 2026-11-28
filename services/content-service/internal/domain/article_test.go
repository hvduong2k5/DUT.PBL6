package domain_test

import (
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/tests/testhelper"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInvalidTransitions(t *testing.T) {
	for _, tc := range []struct {
		status domain.Status
		action string
	}{{domain.Draft, "approve"}, {domain.Draft, "reject"}, {domain.Pending, "publish"}, {domain.Approved, "submit"}, {domain.Rejected, "publish"}, {domain.Published, "approve"}, {domain.Published, "publish"}, {domain.Draft, "hide"}, {domain.Draft, "restore"}, {domain.Published, "restore"}} {
		t.Run(string(tc.status)+"/"+tc.action, func(t *testing.T) {
			id := uuid.New()
			a := domain.Article{CurrentRevisionID: id, Revisions: []domain.Revision{{ID: id, Status: tc.status}}}
			err := a.Transition(tc.action, "reason", testhelper.Full(), time.Now())
			require.ErrorIs(t, err, domain.ErrState)
			require.Empty(t, a.Events, "a rejected transition must not leave an audit event")
		})
	}
	id := uuid.New()
	a := domain.Article{CurrentRevisionID: id, Revisions: []domain.Revision{{ID: id, Status: domain.Draft}}}
	require.ErrorIs(t, a.Transition("delete", "", testhelper.Full(), time.Now()), domain.ErrValidation)
}
func TestPublishAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name  string
		actor domain.Identity
		want  error
	}{
		{"full access", testhelper.Full(), nil},
		{"approver", testhelper.Approver(), nil},
		{"publish grant only", testhelper.Actor(domain.PermPublish), nil},
		{"editor", testhelper.Editor(), domain.ErrForbidden},
		{"no grants", testhelper.Customer(), domain.ErrForbidden},
		{"non global scope", domain.Identity{ID: uuid.New(), Grants: []domain.Grant{{Permission: domain.PermPublish, ScopeType: "CATEGORY", ScopeID: "heritage"}}}, domain.ErrForbidden},
		{"missing identity", domain.Identity{Grants: testhelper.Full().Grants}, domain.ErrUnauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.New()
			a := domain.Article{CurrentRevisionID: id, Revisions: []domain.Revision{{ID: id, Status: domain.Approved}}}
			err := a.Transition("publish", "", tc.actor, time.Now())
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				require.Equal(t, domain.Approved, a.Current().Status)
				require.Nil(t, a.PublishedRevisionID)
				require.False(t, a.Visible)
				return
			}
			require.NoError(t, err)
			require.Equal(t, domain.Published, a.Current().Status)
			require.Equal(t, &id, a.PublishedRevisionID)
			require.True(t, a.Visible)
		})
	}
}

// An ADMIN role name is not an input to authorization at all: only grants are.
func TestPermissionsAreTheOnlyAuthority(t *testing.T) {
	require.NoError(t, testhelper.Actor(domain.PermView).Authorize(domain.PermView))
	require.ErrorIs(t, testhelper.Actor(domain.PermView).Authorize(domain.PermEdit), domain.ErrForbidden)
	require.ErrorIs(t, testhelper.Customer().Authorize(domain.PermView), domain.ErrForbidden)
	require.ErrorIs(t, domain.Identity{}.Authorize(domain.PermView), domain.ErrUnauthenticated)
	require.False(t, testhelper.Customer().Staff())
	require.True(t, testhelper.Actor(domain.PermView).Staff())
	require.False(t, domain.Identity{Grants: testhelper.Full().Grants}.Staff())
	for action, want := range map[string]domain.Permission{"submit": domain.PermSubmit, "approve": domain.PermApprove, "reject": domain.PermReject, "publish": domain.PermPublish, "restore": domain.PermPublish, "hide": domain.PermHide} {
		got, ok := domain.ActionPermission(action)
		require.True(t, ok)
		require.Equal(t, want, got)
	}
	_, ok := domain.ActionPermission("delete")
	require.False(t, ok)
}

func TestHideRestoreAndAudit(t *testing.T) {
	now := time.Now()
	actor := testhelper.Full()
	a, err := domain.NewArticle(testhelper.Content("audit"), actor, now)
	require.NoError(t, err)
	_, err = domain.NewArticle(testhelper.Content("audit"), testhelper.Editor(), now)
	require.NoError(t, err)
	_, err = domain.NewArticle(testhelper.Content("audit"), testhelper.Approver(), now)
	require.ErrorIs(t, err, domain.ErrForbidden)
	for _, action := range []string{"submit", "approve", "publish"} {
		require.NoError(t, a.Transition(action, "", actor, now))
	}
	published := *a.PublishedRevisionID
	require.ErrorIs(t, a.Transition("restore", "", actor, now), domain.ErrState, "a visible article cannot be restored")
	require.NoError(t, a.Transition("hide", "  outdated  ", actor, now))
	require.False(t, a.Visible)
	require.Equal(t, &published, a.PublishedRevisionID, "hide keeps the approved snapshot")
	require.ErrorIs(t, a.Transition("hide", "", actor, now), domain.ErrState)
	require.NoError(t, a.Transition("restore", "", actor, now))
	require.True(t, a.Visible)
	require.Equal(t, domain.Published, a.Revision(published).Status)
	var actions []string
	for _, e := range a.Events {
		actions = append(actions, e.Action)
		require.Equal(t, actor.ID, e.ActorID)
	}
	require.Equal(t, []string{"create", "submit", "approve", "publish", "hide", "restore"}, actions)
	require.Equal(t, "outdated", a.Events[4].Reason)
	require.Equal(t, published, a.Events[4].RevisionID)
	// Editing records the new draft revision and still leaves the public one alone.
	c := testhelper.Content("audit")
	c.Body = "second"
	require.NoError(t, a.Edit(c, actor, now))
	last := a.Events[len(a.Events)-1]
	require.Equal(t, "edit", last.Action)
	require.Equal(t, a.CurrentRevisionID, last.RevisionID)
	require.Equal(t, published, *a.PublishedRevisionID)
}

func TestContentSecurityValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Content)
	}{
		{"empty title", func(c *domain.Content) { c.Title = " " }},
		{"empty body", func(c *domain.Content) { c.Body = "" }},
		{"canonical scheme", func(c *domain.Content) { c.SEO.CanonicalURL = "javascript:alert(1)" }},
		{"schema array", func(c *domain.Content) { c.SEO.Schema = []byte(`[]`) }},
		{"schema null", func(c *domain.Content) { c.SEO.Schema = []byte(`null`) }},
		{"unsafe media", func(c *domain.Content) {
			c.Media = []domain.MediaReference{{ID: uuid.New(), URL: "data:x", Kind: "image"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testhelper.Content("safe")
			tc.change(&c)
			require.ErrorIs(t, c.Validate(), domain.ErrValidation)
		})
	}
}
