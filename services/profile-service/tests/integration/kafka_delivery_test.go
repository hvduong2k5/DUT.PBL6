package integration_test

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

// Executes against the isolated Compose stack, including its real outbox worker.
func TestRealKafkaOutboxDelivery(t *testing.T) {
	url, broker := os.Getenv("TEST_PROFILE_DATABASE_URL"), os.Getenv("TEST_PROFILE_KAFKA_BROKER")
	if url == "" || broker == "" {
		t.Skip("requires isolated Compose database, Kafka and running API worker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	repo := repository.NewCustomerRepository(pool)
	id := uuid.New()
	now := time.Now().UTC()
	require.NoError(t, repo.Create(ctx, &domain.CustomerProfile{ID: id, UserID: uuid.New(), FullName: "Kafka Test", PhoneNumber: "0905111111", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now}))
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM customer_profiles WHERE id=$1`, id) }()
	_, err = repo.UpdateProfileWithOptimisticLock(ctx, id, "Kafka Updated", "0905111111", "", 1)
	require.NoError(t, err)
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{broker}, Topic: "profile.events.v1", Partition: 0, MinBytes: 1, MaxBytes: 1 << 20, MaxWait: time.Second, StartOffset: kafka.FirstOffset})
	defer reader.Close()
	for {
		message, err := reader.ReadMessage(ctx)
		require.NoError(t, err)
		var event struct {
			SpecVersion string `json:"specversion"`
			ID          string `json:"id"`
			Type        string `json:"type"`
			Data        struct {
				CustomerID string `json:"customer_id"`
				NewVersion int    `json:"new_version"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(message.Value, &event))
		if event.Data.CustomerID != id.String() {
			continue
		}
		require.Equal(t, "1.0", event.SpecVersion)
		require.Equal(t, "vn.omama.profile.updated.v1", event.Type)
		require.Equal(t, 2, event.Data.NewVersion)
		require.Equal(t, id.String(), string(message.Key))
		_, err = uuid.Parse(event.ID)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			var published bool
			err := pool.QueryRow(ctx, `SELECT published_at IS NOT NULL FROM outbox_events WHERE id=$1`, event.ID).Scan(&published)
			return err == nil && published
		}, 5*time.Second, 20*time.Millisecond)
		t.Logf("Verified real Kafka CloudEvent type=%s, version=%d and persisted publication acknowledgement", event.Type, event.Data.NewVersion)
		break
	}
}
