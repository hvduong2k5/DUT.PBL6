package persistence

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type uncertainPublisher struct {
	messages []EventMessage
	fail     bool
}

func (p *uncertainPublisher) Publish(_ context.Context, m EventMessage) error {
	p.messages = append(p.messages, m)
	if p.fail {
		return errors.New("synthetic broker ACK lost")
	}
	return nil
}
func TestTargetOutboxUnknownACKKeepsStableBytesAndSequence(t *testing.T) {
	h, _, a, _ := paymentFixture(t)
	publisher := &uncertainPublisher{fail: true}
	if worked, e := h.store.PublishOne(h.ctx, publisher); !worked || e == nil {
		t.Fatal(worked, e)
	}
	if h.count(t, "SELECT count(*) FROM outbox_events WHERE publish_status='PUBLISHED'") != 0 {
		t.Fatal("unknown broker effect acknowledged locally")
	}
	if _, e := h.db.Exec(h.ctx, "UPDATE event_streams SET run_after=clock_timestamp() WHERE aggregate_id=$1", a.ID()); e != nil {
		t.Fatal(e)
	}
	publisher.fail = false
	if worked, e := h.store.PublishOne(h.ctx, publisher); !worked || e != nil {
		t.Fatal(worked, e)
	}
	if len(publisher.messages) != 2 || publisher.messages[0].ID != publisher.messages[1].ID || publisher.messages[0].Sequence != publisher.messages[1].Sequence || !bytes.Equal(publisher.messages[0].Bytes, publisher.messages[1].Bytes) {
		t.Fatal("unknown effect replay changed event identity")
	}
	if h.count(t, "SELECT count(*) FROM event_streams WHERE published_sequence=1") != 1 {
		t.Fatal("stream head not acknowledged once")
	}
}
