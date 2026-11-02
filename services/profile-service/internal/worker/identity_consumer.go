package worker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	kafkaGo "github.com/segmentio/kafka-go"
)

// IdentityProvisioner is the slice of the customer usecase the consumer needs.
type IdentityProvisioner interface {
	ProvisionFromRegistration(ctx context.Context, principalID uuid.UUID, email, firstName, lastName string) (bool, error)
	SuspendByUserID(ctx context.Context, userID uuid.UUID) error
	ReactivateByUserID(ctx context.Context, userID uuid.UUID) error
}

type identityEventKind string

const (
	identityEventRegistered  identityEventKind = "registered"
	identityEventSuspended   identityEventKind = "suspended"
	identityEventReactivated identityEventKind = "reactivated"
	identityEventUnknown     identityEventKind = "unknown"
)

type identityEvent struct {
	Kind        identityEventKind
	PrincipalID uuid.UUID
	Email       string
	FirstName   string
	LastName    string
}

// decodeIdentityEvent classifies one message from the identity events topic.
//
// Identity's outbox publisher sends the bare payload JSON as the Kafka value
// (no envelope and no event-type field), and registrations and lifecycle
// changes all share that topic, so the kind is inferred from the payload
// shape: suspendedAt -> suspended, reactivatedAt -> reactivated, and an
// email/name payload -> registered. Anything else is reported as unknown and
// skipped by the caller. A malformed payload, or one without a valid
// principalId, is an error: it can never be handled, so the consumer drops it
// instead of blocking the partition behind it.
func decodeIdentityEvent(value []byte) (identityEvent, error) {
	var payload struct {
		PrincipalID   *string `json:"principalId"`
		Email         *string `json:"email"`
		FirstName     *string `json:"firstName"`
		LastName      *string `json:"lastName"`
		SuspendedAt   *string `json:"suspendedAt"`
		ReactivatedAt *string `json:"reactivatedAt"`
	}
	if err := json.Unmarshal(value, &payload); err != nil {
		return identityEvent{}, err
	}
	if payload.PrincipalID == nil {
		return identityEvent{}, errors.New("identity event missing principalId")
	}
	id, err := uuid.Parse(*payload.PrincipalID)
	if err != nil || id == uuid.Nil {
		return identityEvent{}, errors.New("identity event has invalid principalId")
	}

	ev := identityEvent{Kind: identityEventUnknown, PrincipalID: id}
	switch {
	case payload.SuspendedAt != nil:
		ev.Kind = identityEventSuspended
	case payload.ReactivatedAt != nil:
		ev.Kind = identityEventReactivated
	case payload.Email != nil || payload.FirstName != nil || payload.LastName != nil:
		ev.Kind = identityEventRegistered
		if payload.Email != nil {
			ev.Email = *payload.Email
		}
		if payload.FirstName != nil {
			ev.FirstName = *payload.FirstName
		}
		if payload.LastName != nil {
			ev.LastName = *payload.LastName
		}
	}
	return ev, nil
}

// IdentityConsumer provisions and maintains customer profiles from Identity
// lifecycle events, closing the gap where a registered customer existed in
// Identity/Keycloak but had no profile row for Order to resolve.
type IdentityConsumer struct {
	brokers     []string
	topic       string
	groupID     string
	provisioner IdentityProvisioner

	// topicWait bounds how long Start waits for the identity topic to exist
	// before joining the consumer group (see waitForTopic).
	topicWait time.Duration

	mu       sync.Mutex
	reader   *kafkaGo.Reader
	stopChan chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewIdentityConsumer(brokers, topic, groupID string, provisioner IdentityProvisioner) *IdentityConsumer {
	return &IdentityConsumer{
		brokers:     strings.Split(brokers, ","),
		topic:       topic,
		groupID:     groupID,
		provisioner: provisioner,
		topicWait:   2 * time.Minute,
		stopChan:    make(chan struct{}),
	}
}

func (c *IdentityConsumer) newReader() *kafkaGo.Reader {
	return kafkaGo.NewReader(kafkaGo.ReaderConfig{
		Brokers:  c.brokers,
		Topic:    c.topic,
		GroupID:  c.groupID,
		MinBytes: 1,
		MaxBytes: 1 << 20,
		MaxWait:  2 * time.Second,
		// Synchronous commits: an offset advances only after its event has
		// been handled, so a crash cannot silently lose a registration.
		CommitInterval: 0,
		// A fresh consumer group starts from the earliest offset so
		// registrations published before this consumer first ran are still
		// provisioned; handling is idempotent, so replays are harmless.
		StartOffset: kafkaGo.FirstOffset,
	})
}

// waitForTopic blocks until the identity topic exists on the broker. A
// kafka-go group reader resolves its partitions when it joins the group and
// does not reliably pick up a topic that is only created afterwards (the
// topic appears with the first-ever registration), which would silently
// stall provisioning until the next restart or rebalance. Waiting for the
// topic up front removes that failure mode on first deployment. The wait is
// bounded: after the deadline the consumer starts anyway and keeps the
// pre-existing behaviour rather than blocking the service forever.
func (c *IdentityConsumer) waitForTopic(ctx context.Context) {
	deadline := time.Now().Add(c.topicWait)
	for {
		conn, err := kafkaGo.Dial("tcp", c.brokers[0])
		if err == nil {
			partitions, err := conn.ReadPartitions(c.topic)
			_ = conn.Close()
			if err == nil && len(partitions) > 0 {
				return
			}
		}
		if time.Now().After(deadline) {
			log.Warn().Str("topic", c.topic).Dur("waited", c.topicWait).Msg("identity consumer: topic not visible yet; starting anyway")
			return
		}
		if !c.backoff(ctx, 5*time.Second) {
			return
		}
	}
}

// handle applies one event value. Malformed and unknown events are dropped
// (nil error) so they are committed and never redelivered; a provisioning
// failure returns the error so the caller leaves the offset uncommitted.
func (c *IdentityConsumer) handle(ctx context.Context, value []byte) error {
	ev, err := decodeIdentityEvent(value)
	if err != nil {
		log.Warn().Err(err).Msg("identity consumer: dropping malformed identity event")
		return nil
	}
	switch ev.Kind {
	case identityEventRegistered:
		created, err := c.provisioner.ProvisionFromRegistration(ctx, ev.PrincipalID, ev.Email, ev.FirstName, ev.LastName)
		if err != nil {
			return err
		}
		if created {
			log.Info().Str("user_id", ev.PrincipalID.String()).Msg("identity consumer: provisioned customer profile from registration")
		}
		return nil
	case identityEventSuspended:
		return c.provisioner.SuspendByUserID(ctx, ev.PrincipalID)
	case identityEventReactivated:
		return c.provisioner.ReactivateByUserID(ctx, ev.PrincipalID)
	default:
		log.Debug().Str("user_id", ev.PrincipalID.String()).Msg("identity consumer: skipping unknown identity event shape")
		return nil
	}
}

func (c *IdentityConsumer) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.waitForTopic(ctx)
		if ctx.Err() != nil || c.stopped() {
			return
		}
		reader := c.newReader()
		c.mu.Lock()
		c.reader = reader
		c.mu.Unlock()
		defer reader.Close()
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil || c.stopped() {
					return
				}
				log.Error().Err(err).Msg("identity consumer: fetch failed; retrying")
				if !c.backoff(ctx, 5*time.Second) {
					return
				}
				continue
			}

			handleCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err = c.handle(handleCtx, msg.Value)
			cancel()
			if err != nil {
				// The offset is deliberately left uncommitted so the event
				// is redelivered on a later session; back off so a failing
				// dependency does not spin the loop meanwhile.
				log.Error().Err(err).Msg("identity consumer: handling failed; offset not committed")
				if !c.backoff(ctx, 2*time.Second) {
					return
				}
				continue
			}
			if err := reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil && !c.stopped() {
				log.Error().Err(err).Msg("identity consumer: commit failed")
			}
		}
	}()
}

func (c *IdentityConsumer) stopped() bool {
	select {
	case <-c.stopChan:
		return true
	default:
		return false
	}
}

// backoff waits for d, returning false when shutdown interrupts the wait.
func (c *IdentityConsumer) backoff(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-c.stopChan:
		return false
	case <-time.After(d):
		return true
	}
}

func (c *IdentityConsumer) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
		c.mu.Lock()
		if c.reader != nil {
			_ = c.reader.Close()
		}
		c.mu.Unlock()
	})
	c.wg.Wait()
}
