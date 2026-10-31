package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDecodeIdentityEvent(t *testing.T) {
	principal := uuid.New()

	tests := []struct {
		name     string
		value    string
		wantKind identityEventKind
		wantErr  bool
	}{
		{
			name:     "registered event from RegistrationOrchestrator payload",
			value:    `{"principalId":"` + principal.String() + `","email":"khach@example.vn","firstName":"Van","lastName":"Nguyen"}`,
			wantKind: identityEventRegistered,
		},
		{
			name:     "suspended event from UserLifecycle payload",
			value:    `{"principalId":"` + principal.String() + `","reason":"Administrative suspension","suspendedAt":"2026-10-30T01:00:00Z"}`,
			wantKind: identityEventSuspended,
		},
		{
			name:     "reactivated event from UserLifecycle payload",
			value:    `{"principalId":"` + principal.String() + `","reactivatedAt":"2026-10-30T01:00:00Z"}`,
			wantKind: identityEventReactivated,
		},
		{
			name:     "unknown shape with valid principal is skipped, not an error",
			value:    `{"principalId":"` + principal.String() + `","somethingElse":true}`,
			wantKind: identityEventUnknown,
		},
		{
			name:    "missing principalId is malformed",
			value:   `{"email":"khach@example.vn"}`,
			wantErr: true,
		},
		{
			name:    "non-uuid principalId is malformed",
			value:   `{"principalId":"not-a-uuid","email":"khach@example.vn"}`,
			wantErr: true,
		},
		{
			name:    "nil principalId is malformed",
			value:   `{"principalId":"00000000-0000-0000-0000-000000000000","email":"khach@example.vn"}`,
			wantErr: true,
		},
		{
			name:    "invalid json is malformed",
			value:   `{not json`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := decodeIdentityEvent([]byte(tc.value))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantKind, ev.Kind)
			require.Equal(t, principal, ev.PrincipalID)
		})
	}
}

func TestDecodeIdentityEventRegisteredFields(t *testing.T) {
	principal := uuid.New()
	ev, err := decodeIdentityEvent([]byte(`{"principalId":"` + principal.String() + `","email":"khach@example.vn","firstName":"Van","lastName":"Nguyen"}`))
	require.NoError(t, err)
	require.Equal(t, identityEventRegistered, ev.Kind)
	require.Equal(t, "khach@example.vn", ev.Email)
	require.Equal(t, "Van", ev.FirstName)
	require.Equal(t, "Nguyen", ev.LastName)
}

// With an unreachable broker the topic wait must give up after its deadline
// instead of blocking startup forever (the consumer then starts anyway).
func TestIdentityConsumerWaitForTopicTimesOut(t *testing.T) {
	c := NewIdentityConsumer("127.0.0.1:1", "identity.events.v1", "test-group", &fakeProvisioner{})
	c.topicWait = 300 * time.Millisecond
	start := time.Now()
	c.waitForTopic(context.Background())
	require.Less(t, time.Since(start), 10*time.Second)
}

type fakeProvisioner struct {
	registered   []uuid.UUID
	suspended    []uuid.UUID
	reactivated  []uuid.UUID
	provisionErr error
}

func (f *fakeProvisioner) ProvisionFromRegistration(_ context.Context, principalID uuid.UUID, _, _, _ string) (bool, error) {
	if f.provisionErr != nil {
		return false, f.provisionErr
	}
	f.registered = append(f.registered, principalID)
	return true, nil
}

func (f *fakeProvisioner) SuspendByUserID(_ context.Context, userID uuid.UUID) error {
	f.suspended = append(f.suspended, userID)
	return nil
}

func (f *fakeProvisioner) ReactivateByUserID(_ context.Context, userID uuid.UUID) error {
	f.reactivated = append(f.reactivated, userID)
	return nil
}

func TestIdentityConsumerHandle(t *testing.T) {
	principal := uuid.New()
	registered := `{"principalId":"` + principal.String() + `","email":"khach@example.vn","firstName":"Van","lastName":"Nguyen"}`

	t.Run("registered event provisions the profile", func(t *testing.T) {
		fake := &fakeProvisioner{}
		c := &IdentityConsumer{provisioner: fake}
		require.NoError(t, c.handle(context.Background(), []byte(registered)))
		require.Equal(t, []uuid.UUID{principal}, fake.registered)
	})

	t.Run("suspended and reactivated events reach the provisioner", func(t *testing.T) {
		fake := &fakeProvisioner{}
		c := &IdentityConsumer{provisioner: fake}
		require.NoError(t, c.handle(context.Background(), []byte(`{"principalId":"`+principal.String()+`","suspendedAt":"2026-10-30T01:00:00Z"}`)))
		require.NoError(t, c.handle(context.Background(), []byte(`{"principalId":"`+principal.String()+`","reactivatedAt":"2026-10-30T01:00:00Z"}`)))
		require.Equal(t, []uuid.UUID{principal}, fake.suspended)
		require.Equal(t, []uuid.UUID{principal}, fake.reactivated)
	})

	t.Run("provisioning failure is returned so the offset stays uncommitted", func(t *testing.T) {
		fake := &fakeProvisioner{provisionErr: errors.New("database unavailable")}
		c := &IdentityConsumer{provisioner: fake}
		require.Error(t, c.handle(context.Background(), []byte(registered)))
	})

	t.Run("malformed and unknown events are dropped without error", func(t *testing.T) {
		fake := &fakeProvisioner{}
		c := &IdentityConsumer{provisioner: fake}
		require.NoError(t, c.handle(context.Background(), []byte(`{not json`)))
		require.NoError(t, c.handle(context.Background(), []byte(`{"principalId":"`+principal.String()+`","somethingElse":true}`)))
		require.Empty(t, fake.registered)
	})
}
