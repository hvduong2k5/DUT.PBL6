package kafka

import (
	"context"
	"fmt"
	segmentio "github.com/segmentio/kafka-go"
	"net/http"
	"sync/atomic"
	"time"
)

var procurementHealthy atomic.Bool
var procurementRetries atomic.Int64
var procurementDeadLetters atomic.Int64
var procurementLag atomic.Int64

// ProcurementMonitor is an internal operations listener. Bind to a private
// interface/network; no business data or payloads are exposed here.
func ProcurementMonitor(enabled bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if enabled && !procurementHealthy.Load() {
			http.Error(w, "procurement consumer unhealthy", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ready")
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		healthy := 0
		if procurementHealthy.Load() {
			healthy = 1
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "inventory_procurement_healthy %d\ninventory_procurement_retries_total %d\ninventory_procurement_dlq_total %d\ninventory_procurement_consumer_lag %d\n", healthy, procurementRetries.Load(), procurementDeadLetters.Load(), procurementLag.Load())
	})
	return mux
}

const procurementGroup = "inventory-service-procurement-goods-received-group"

func monitorProcurementLag(ctx context.Context, brokers []string) {
	defer func() {
		if recover() != nil {
			procurementLag.Store(-1)
		}
	}()
	procurementLag.Store(-1)
	client := &segmentio.Client{Addr: segmentio.TCP(brokers...), Timeout: 5 * time.Second}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		sampleCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		lag, err := readProcurementLag(sampleCtx, client)
		cancel()
		if err != nil {
			lag = -1
		}
		procurementLag.Store(lag)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func readProcurementLag(ctx context.Context, client *segmentio.Client) (int64, error) {
	meta, err := client.Metadata(ctx, &segmentio.MetadataRequest{Topics: []string{procurementTopic}})
	if err != nil {
		return -1, err
	}
	var partitions []int
	var offsets []segmentio.OffsetRequest
	for _, topic := range meta.Topics {
		if topic.Error != nil {
			return -1, topic.Error
		}
		for _, p := range topic.Partitions {
			if p.Error != nil {
				return -1, p.Error
			}
			partitions = append(partitions, p.ID)
			offsets = append(offsets, segmentio.LastOffsetOf(p.ID))
		}
	}
	if len(partitions) == 0 {
		return -1, fmt.Errorf("no Procurement partitions")
	}
	committed, err := client.OffsetFetch(ctx, &segmentio.OffsetFetchRequest{GroupID: procurementGroup, Topics: map[string][]int{procurementTopic: partitions}})
	if err != nil {
		return -1, err
	}
	if committed.Error != nil {
		return -1, committed.Error
	}
	end, err := client.ListOffsets(ctx, &segmentio.ListOffsetsRequest{Topics: map[string][]segmentio.OffsetRequest{procurementTopic: offsets}})
	if err != nil {
		return -1, err
	}
	byPartition := map[int]int64{}
	for _, p := range committed.Topics[procurementTopic] {
		if p.Error != nil {
			return -1, p.Error
		}
		if p.CommittedOffset < 0 {
			return -1, fmt.Errorf("group has not committed all partitions")
		}
		byPartition[p.Partition] = p.CommittedOffset
	}
	if len(end.Topics[procurementTopic]) != len(partitions) || len(byPartition) != len(partitions) {
		return -1, fmt.Errorf("incomplete lag sample")
	}
	var lag int64
	for _, p := range end.Topics[procurementTopic] {
		if p.Error != nil {
			return -1, p.Error
		}
		offset, ok := byPartition[p.Partition]
		if !ok {
			return -1, fmt.Errorf("missing committed offset")
		}
		if p.LastOffset > offset {
			lag += p.LastOffset - offset
		}
	}
	return lag, nil
}
