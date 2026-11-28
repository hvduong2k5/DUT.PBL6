package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (s *Profiles) FindBatch(ctx context.Context, namespace, batch string) (domain.Profile, error) {
	var id string
	err := s.DB.QueryRow(ctx, `SELECT id::text FROM traceability_profiles WHERE source_namespace=$1 AND batch_id=$2`, namespace, batch).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Profile{}, err
	}
	return s.Get(ctx, id)
}

// A single statement provides one consistent snapshot and keeps corrupt rows
// separate from a failed database read. Neither path calls another service.
func (s *Profiles) ResolveProfiles(ctx context.Context, bindings []domain.LinkBinding) ([]domain.ResolutionProfile, error) {
	namespaces, batches, skus := []string{}, []string{}, []string{}
	for _, b := range bindings {
		namespaces = append(namespaces, b.SourceNamespace)
		batches = append(batches, b.BatchID)
		skus = append(skus, b.SKUCode)
	}
	rows, err := s.DB.Query(ctx, `WITH bindings AS (SELECT DISTINCT namespace,batch,sku FROM unnest($1::text[],$2::text[],$3::text[]) AS q(namespace,batch,sku))
 SELECT q.namespace,q.batch,q.sku,to_jsonb(p)||jsonb_build_object('revisions',COALESCE((SELECT jsonb_agg(to_jsonb(r)-'profile_id') FROM traceability_revisions r WHERE r.profile_id=p.id AND r.number=p.public_revision),'[]'::jsonb))
 FROM bindings q JOIN traceability_profiles p ON p.source_namespace=q.namespace AND p.batch_id=q.batch AND p.sku_code=q.sku`, namespaces, batches, skus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.ResolutionProfile, len(bindings))
	found := map[domain.LinkBinding]domain.ResolutionProfile{}
	for rows.Next() {
		var binding domain.LinkBinding
		var data []byte
		if err = rows.Scan(&binding.SourceNamespace, &binding.BatchID, &binding.SKUCode, &data); err != nil {
			return nil, err
		}
		var profile domain.Profile
		if json.Unmarshal(data, &profile) != nil {
			found[binding] = domain.ResolutionProfile{Unavailable: true}
			continue
		}
		found[binding] = domain.ResolutionProfile{Profile: &profile}
	}
	for i, binding := range bindings {
		result[i] = found[binding]
	}
	return result, rows.Err()
}
