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
	}{{domain.Draft, "approve"}, {domain.Draft, "reject"}, {domain.Pending, "publish"}, {domain.Approved, "submit"}, {domain.Rejected, "publish"}, {domain.Published, "approve"}, {domain.Published, "publish"}, {domain.Draft, "hide"}} {
		t.Run(string(tc.status)+"/"+tc.action, func(t *testing.T) {
			id := uuid.New()
			a := domain.Article{CurrentRevisionID: id, Revisions: []domain.Revision{{ID: id, Status: tc.status}}}
			err := a.Transition(tc.action, "reason", testhelper.Actor("ADMIN"), time.Now())
			require.ErrorIs(t, err, domain.ErrState)
		})
	}
}
func TestPublishAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name  string
		actor domain.Identity
		want  error
	}{
		{"admin", testhelper.Actor("ADMIN"), nil},
		{"approver", testhelper.Actor("CONTENT_APPROVER"), nil},
		{"editor", testhelper.Actor("CONTENT_EDITOR"), domain.ErrForbidden},
		{"other role", testhelper.Actor("CUSTOMER"), domain.ErrForbidden},
		{"missing identity", domain.Identity{}, domain.ErrUnauthenticated},
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
