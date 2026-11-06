package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"runtime/debug"
	segmentiokafka "github.com/segmentio/kafka-go"
)

var (
	ErrListenerAlreadyRunning = errors.New("kafka listener is already running")
	ErrListenerNotRunning     = errors.New("kafka listener is not running")
	ErrInvalidEventPayload    = errors.New("invalid event payload")
)

// KafkaMessage models a decoupled Kafka message.
type KafkaMessage struct {
	Topic     string
	Key       []byte
	Value     []byte
	Offset    int64
	Partition int
}

// MessageReader abstracts the consumer reader for testability.
type MessageReader interface {
	FetchMessage(ctx context.Context) (KafkaMessage, error)
	CommitMessages(ctx context.Context, msgs ...KafkaMessage) error
	Close() error
}

// SegmentioReaderAdapter implements MessageReader using segmentio/kafka-go.Reader.
type SegmentioReaderAdapter struct {
	reader *segmentiokafka.Reader
}

// NewSegmentioReaderAdapter constructs an adapter wrapping segmentio/kafka-go.Reader.
func NewSegmentioReaderAdapter(brokers []string, topic, groupID string) *SegmentioReaderAdapter {
	r := segmentiokafka.NewReader(segmentiokafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})
	return &SegmentioReaderAdapter{reader: r}
}

func (a *SegmentioReaderAdapter) FetchMessage(ctx context.Context) (KafkaMessage, error) {
	msg, err := a.reader.FetchMessage(ctx)
	if err != nil {
		return KafkaMessage{}, err
	}
	return KafkaMessage{
		Topic:     msg.Topic,
		Key:       msg.Key,
		Value:     msg.Value,
		Offset:    msg.Offset,
		Partition: msg.Partition,
	}, nil
}

func (a *SegmentioReaderAdapter) CommitMessages(ctx context.Context, msgs ...KafkaMessage) error {
	segMsgs := make([]segmentiokafka.Message, len(msgs))
	for i, m := range msgs {
		segMsgs[i] = segmentiokafka.Message{
			Topic:     m.Topic,
			Key:       m.Key,
			Value:     m.Value,
			Offset:    m.Offset,
			Partition: m.Partition,
		}
	}
	return a.reader.CommitMessages(ctx, segMsgs...)
}

func (a *SegmentioReaderAdapter) Close() error {
	if a.reader != nil {
		return a.reader.Close()
	}
	return nil
}

// OrderPaidHandler processes vn.omama.order.paid.v1 events from order.events.v1.
type OrderPaidHandler struct {
	commitStockUC   *usecase.CommitStockDeductionUseCase
	idempotencyRepo port.IdempotencyRepository
}

// NewOrderPaidHandler creates a new OrderPaidHandler.
func NewOrderPaidHandler(
	commitStockUC *usecase.CommitStockDeductionUseCase,
	idempotencyRepo ...port.IdempotencyRepository,
) *OrderPaidHandler {
	var idemp port.IdempotencyRepository
	if len(idempotencyRepo) > 0 {
		idemp = idempotencyRepo[0]
	}
	return &OrderPaidHandler{
		commitStockUC:   commitStockUC,
		idempotencyRepo: idemp,
	}
}

type cloudEventEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type orderPaidData struct {
	OrderID string `json:"order_id"`
}

// Handle processes one message, guaranteeing idempotency and stock deduction commit.
func (h *OrderPaidHandler) Handle(ctx context.Context, msg KafkaMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[OrderPaidHandler Panic Recovery] Panic recovered for message on topic %s (key %s): %v\nStack trace:\n%s",
				msg.Topic, string(msg.Key), r, debug.Stack())
			err = fmt.Errorf("panic recovered in OrderPaidHandler: %v", r)
		}
	}()

	if len(msg.Value) == 0 {
		return nil
	}

	var env cloudEventEnvelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		log.Printf("[OrderPaidHandler] Failed to parse message JSON: %v", err)
		return ErrInvalidEventPayload
	}

	// Filter event type: only handle vn.omama.order.paid.v1
	if env.Type != entity.EventTypeOrderPaid {
		// Ignore events meant for other handlers on the topic
		return nil
	}

	var data orderPaidData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		log.Printf("[OrderPaidHandler] Failed to parse OrderPaidData: %v", err)
		return fmt.Errorf("%w: invalid order paid data", ErrInvalidEventPayload)
	}

	orderID := strings.TrimSpace(data.OrderID)
	if orderID == "" {
		// Fallback: check string partition key
		keyStr := strings.TrimSpace(string(msg.Key))
		if strings.HasPrefix(keyStr, "order_id:") {
			orderID = strings.TrimPrefix(keyStr, "order_id:")
		} else if keyStr != "" {
			orderID = keyStr
		}
	}

	if orderID == "" {
		return fmt.Errorf("%w: empty order_id", ErrInvalidEventPayload)
	}

	// 1. Execute CommitStockDeductionUseCase first.
	// CommitStockDeductionUseCase is inherently idempotent at domain/DB level
	// (if reservation is already COMMITTED, it returns nil).
	err = h.commitStockUC.Execute(ctx, orderID)
	if err != nil {
		if errors.Is(err, entity.ErrReservationAlreadyProcessed) {
			log.Printf("[OrderPaidHandler] Reservation for order %s was already processed", orderID)
		} else {
			log.Printf("[OrderPaidHandler] CommitStockDeductionUseCase failed for order %s: %v", orderID, err)
			return err
		}
	} else {
		log.Printf("[OrderPaidHandler] Successfully committed stock deduction for order %s", orderID)
	}

	// 2. Record idempotency key ONLY after successful execution (or already processed).
	// This prevents premature idempotency registration that leads to lost stock deductions on Kafka retries.
	if h.idempotencyRepo != nil {
		idempKey := fmt.Sprintf("idemp:kafka:order.events.v1:order_paid:%s", orderID)
		if _, idempErr := h.idempotencyRepo.CheckOrSet(ctx, nil, idempKey); idempErr != nil {
			log.Printf("[OrderPaidHandler] Idempotency recording error for order %s: %v", orderID, idempErr)
			return idempErr
		}
	}

	return nil
}

// KafkaConsumerListener manages continuous consumption and commit loop.
type KafkaConsumerListener struct {
	reader    MessageReader
	handler   *OrderPaidHandler
	isRunning atomic.Bool
	stopChan  chan struct{}
	doneChan  chan struct{}
}

// NewKafkaConsumerListener creates a new listener.
func NewKafkaConsumerListener(reader MessageReader, handler *OrderPaidHandler) *KafkaConsumerListener {
	return &KafkaConsumerListener{
		reader:   reader,
		handler:  handler,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start launches the consumer loop in a background goroutine.
func (l *KafkaConsumerListener) Start(ctx context.Context) error {
	if !l.isRunning.CompareAndSwap(false, true) {
		return ErrListenerAlreadyRunning
	}

	l.stopChan = make(chan struct{})
	l.doneChan = make(chan struct{})

	go l.run(ctx)
	return nil
}

// Stop gracefully shuts down the consumer loop.
func (l *KafkaConsumerListener) Stop(timeout ...time.Duration) error {
	if !l.isRunning.CompareAndSwap(true, false) {
		return ErrListenerNotRunning
	}

	close(l.stopChan)

	to := 5 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		to = timeout[0]
	}

	select {
	case <-l.doneChan:
		return nil
	case <-time.After(to):
		return errors.New("timeout waiting for kafka consumer listener to stop")
	}
}

func (l *KafkaConsumerListener) run(ctx context.Context) {
	defer close(l.doneChan)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[KafkaConsumerListener Panic Recovery] Unexpected panic in consumer loop: %v\nStack trace:\n%s",
				r, debug.Stack())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopChan:
			return
		default:
			msg, err := l.reader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				// Backoff briefly on read error
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if handleErr := l.handler.Handle(ctx, msg); handleErr != nil {
				log.Printf("[KafkaConsumerListener] Message processing error on topic %s: %v", msg.Topic, handleErr)
				// Do not commit message on transient failure, allowing Kafka to redeliver
				continue
			}

			if commitErr := l.reader.CommitMessages(ctx, msg); commitErr != nil {
				log.Printf("[KafkaConsumerListener] Failed to commit message offset %d: %v", msg.Offset, commitErr)
			}
		}
	}
}
