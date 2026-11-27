package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

// PriceProjection represents a read-model row in sku_prices_projection for fast cart enrichment.
type PriceProjection struct {
	SKU           string
	Channel       string
	ProductName   string
	UnitPrice     int64
	WeightGrams   int64
	PackagingType string
	IsSaleable    bool
	PriceVersion  int64
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

type catalogPriceChangedEnvelope struct {
	SpecVersion string `json:"specversion"`
	ID          string `json:"id"`
	Source      string `json:"source"`
	Type        string `json:"type"`
	Subject     string `json:"subject"`
	Data        struct {
		ProductID     string `json:"product_id"`
		VariantID     string `json:"variant_id"`
		SKUCode       string `json:"sku_code"`
		PriceRevision int64  `json:"price_revision"`
		ProductName   string `json:"product_name"`
		WeightGrams   int64  `json:"weight_grams"`
		PackagingType string `json:"packaging_type"`
		IsSaleable    *bool  `json:"is_saleable"`
		Prices        []struct {
			Channel       string     `json:"channel"`
			CurrencyCode  string     `json:"currency_code"`
			AmountUnits   int64      `json:"amount_units"`
			AmountNanos   int64      `json:"amount_nanos"`
			EffectiveFrom time.Time  `json:"effective_from"`
			EffectiveTo   *time.Time `json:"effective_to"`
			Status        string     `json:"status"`
			Version       int64      `json:"version"`
			PriceRevision int64      `json:"price_revision"`
		} `json:"prices"`
	} `json:"data"`
}

// ProcessCatalogPriceEvent parses a catalog.events.v1 CloudEvent and upserts sku_prices_projection.
func (s *Service) ProcessCatalogPriceEvent(ctx context.Context, raw []byte) (string, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return "", domain.E("INVALID_CATALOG_EVENT", 422)
	}
	var env catalogPriceChangedEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || env.SpecVersion != "1.0" {
		return "", domain.E("INVALID_CATALOG_EVENT", 422)
	}
	if env.Type != "vn.omama.catalog.product.price.changed.v1" {
		return "IGNORED", nil
	}
	if _, err := uuid.Parse(env.ID); err != nil || strings.TrimSpace(env.Data.SKUCode) == "" || len(env.Data.Prices) == 0 {
		return "", domain.E("INVALID_CATALOG_EVENT", 422)
	}
	if s.DB == nil {
		return "", domain.E("DATABASE_UNAVAILABLE", 503)
	}

	name := strings.TrimSpace(env.Data.ProductName)
	if name == "" {
		name = env.Data.SKUCode
	}
	saleable := true
	if env.Data.IsSaleable != nil {
		saleable = *env.Data.IsSaleable
	}

	for _, entry := range env.Data.Prices {
		if entry.Status != "ACTIVE" || entry.CurrencyCode != "VND" || entry.Channel == "" || entry.AmountUnits < 0 {
			continue
		}
		version := entry.PriceRevision
		if version == 0 {
			version = env.Data.PriceRevision
		}
		if version == 0 {
			version = entry.Version
		}
		effectiveFrom := entry.EffectiveFrom
		if effectiveFrom.IsZero() {
			effectiveFrom = time.Now().UTC()
		}
		proj := PriceProjection{
			SKU:           env.Data.SKUCode,
			Channel:       entry.Channel,
			ProductName:   name,
			UnitPrice:     entry.AmountUnits,
			WeightGrams:   env.Data.WeightGrams,
			PackagingType: env.Data.PackagingType,
			IsSaleable:    saleable,
			PriceVersion:  version,
			EffectiveFrom: effectiveFrom,
			EffectiveTo:   entry.EffectiveTo,
		}
		if err := s.UpsertPriceProjection(ctx, proj); err != nil {
			return "", err
		}
	}
	return "APPLIED", nil
}

// UpsertPriceProjection inserts or updates a SKU price projection guarded by monotonic price_version.
func (s *Service) UpsertPriceProjection(ctx context.Context, p PriceProjection) error {
	if s.DB == nil {
		return domain.E("DATABASE_UNAVAILABLE", 503)
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO sku_prices_projection (
			sku_code, channel, product_name, unit_price, weight_grams,
			packaging_type, is_saleable, price_version, effective_from, effective_to, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (sku_code, channel) DO UPDATE
		SET product_name   = CASE WHEN EXCLUDED.product_name <> '' AND EXCLUDED.product_name <> EXCLUDED.sku_code THEN EXCLUDED.product_name ELSE sku_prices_projection.product_name END,
			unit_price     = EXCLUDED.unit_price,
			weight_grams   = CASE WHEN EXCLUDED.weight_grams > 0 THEN EXCLUDED.weight_grams ELSE sku_prices_projection.weight_grams END,
			packaging_type = CASE WHEN EXCLUDED.packaging_type <> '' THEN EXCLUDED.packaging_type ELSE sku_prices_projection.packaging_type END,
			is_saleable    = EXCLUDED.is_saleable,
			price_version  = EXCLUDED.price_version,
			effective_from = EXCLUDED.effective_from,
			effective_to   = EXCLUDED.effective_to,
			synced_at      = NOW()
		WHERE EXCLUDED.price_version >= sku_prices_projection.price_version`,
		p.SKU, p.Channel, p.ProductName, p.UnitPrice, p.WeightGrams,
		p.PackagingType, p.IsSaleable, p.PriceVersion, p.EffectiveFrom, p.EffectiveTo,
	)
	return err
}

// LookupCartPrices reads display-only cart pricing from sku_prices_projection for the requested SKUs.
func (s *Service) LookupCartPrices(ctx context.Context, channel string, skus []string) (map[string]PriceProjection, error) {
	out := make(map[string]PriceProjection, len(skus))
	if s.DB == nil || len(skus) == 0 {
		return out, nil
	}
	rows, err := s.DB.Query(ctx, `
		SELECT sku_code, product_name, unit_price, weight_grams, packaging_type, is_saleable, price_version
		FROM sku_prices_projection
		WHERE channel = $1
		  AND sku_code = ANY($2)
		  AND effective_from <= NOW()
		  AND (effective_to IS NULL OR effective_to > NOW())`,
		channel, skus,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p PriceProjection
		p.Channel = channel
		if err := rows.Scan(&p.SKU, &p.ProductName, &p.UnitPrice, &p.WeightGrams, &p.PackagingType, &p.IsSaleable, &p.PriceVersion); err != nil {
			return nil, err
		}
		out[p.SKU] = p
	}
	return out, rows.Err()
}

func (s *Service) consumeCatalog(ctx context.Context) {
	defer s.stop.Done()
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        splitBrokers(s.C.Kafka),
		Topic:          "catalog.events.v1",
		GroupID:        consumerGroup(s.C) + "-catalog-v1",
		StartOffset:    kafka.FirstOffset,
		MinBytes:       1,
		MaxBytes:       1 << 20,
		CommitInterval: 0,
	})
	defer reader.Close()
	for ctx.Err() == nil {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			continue
		}
		for ctx.Err() == nil {
			_, err = s.ProcessCatalogPriceEvent(ctx, message.Value)
			if err != nil && !errors.Is(err, context.Canceled) {
				_, code := s.ErrorCode(err)
				if code >= 500 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						continue
					}
				}
				slog.Warn("catalog price event rejected", "code", safeCode(s, err))
			}
			if err = reader.CommitMessages(ctx, message); err == nil {
				break
			}
		}
	}
}
