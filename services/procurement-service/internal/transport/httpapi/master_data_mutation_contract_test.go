package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

func (r *httpRepository) GetMasterDataMutationResult(_ context.Context, actor, key string) (domain.MasterDataMutationResult, error) {
	v, ok := r.masterMutations[actor+"\x1f"+key]
	if !ok {
		return v, domain.ErrNotFound
	}
	return v, nil
}
func (r *httpRepository) MutateMasterData(ctx context.Context, c domain.MasterDataMutation) (domain.MasterDataMutationResult, error) {
	var item any
	now := time.Now().UTC()
	switch c.Action {
	case "EDIT_SUPPLIER", "SUPPLIER_STATUS":
		v, err := r.GetSupplier(ctx, c.ResourceID)
		if err != nil {
			return domain.MasterDataMutationResult{}, err
		}
		if !v.UpdatedAt.Equal(c.ExpectedUpdatedAt) {
			return domain.MasterDataMutationResult{}, domain.ErrConflict
		}
		if c.Action == "EDIT_SUPPLIER" {
			if c.Supplier.Code != v.Code {
				return domain.MasterDataMutationResult{}, domain.ErrConflict
			}
			old := v
			v = *c.Supplier
			v.ID, v.CreatedAt, v.Status = old.ID, old.CreatedAt, old.Status
		} else {
			v.Status = domain.SupplierStatus(c.Status)
			if v.Status == domain.SupplierSuspended {
				for id, m := range r.materials {
					if m.PreferredSupplierID != nil && *m.PreferredSupplierID == v.ID {
						m.PreferredSupplierID = nil
						m.UpdatedAt = now
						r.materials[id] = m
					}
				}
			}
		}
		v.UpdatedAt = now
		r.suppliers[v.ID] = v
		item = v
	case "EDIT_MATERIAL", "MATERIAL_STATUS", "PREFERRED_SUPPLIER":
		v, err := r.GetMaterial(ctx, c.ResourceID)
		if err != nil {
			return domain.MasterDataMutationResult{}, err
		}
		if !v.UpdatedAt.Equal(c.ExpectedUpdatedAt) {
			return domain.MasterDataMutationResult{}, domain.ErrConflict
		}
		switch c.Action {
		case "EDIT_MATERIAL":
			if c.Material.Code != v.Code || c.Material.BaseUnit != v.BaseUnit {
				return domain.MasterDataMutationResult{}, domain.ErrConflict
			}
			v.Name, v.Description = c.Material.Name, c.Material.Description
		case "MATERIAL_STATUS":
			v.Status = domain.MaterialStatus(c.Status)
			if v.Status == domain.MaterialInactive {
				v.PreferredSupplierID = nil
			}
		case "PREFERRED_SUPPLIER":
			if c.PreferredSupplierID == uuid.Nil {
				v.PreferredSupplierID = nil
			} else {
				s, err := r.GetSupplier(ctx, c.PreferredSupplierID)
				if err != nil {
					return domain.MasterDataMutationResult{}, err
				}
				eligible := false
				for _, o := range r.offers[s.ID] {
					eligible = eligible || o.MaterialID == v.ID && o.EffectiveAt(now) && o.Unit == v.BaseUnit
				}
				if s.Status != domain.SupplierActive || v.Status != domain.MaterialActive || !eligible {
					return domain.MasterDataMutationResult{}, domain.ErrInvalidState
				}
				p := s.ID
				v.PreferredSupplierID = &p
			}
		}
		v.UpdatedAt = now
		r.materials[v.ID] = v
		item = v
	case "OFFER_STATUS":
		s, err := r.GetSupplier(ctx, c.SupplierID)
		if err != nil {
			return domain.MasterDataMutationResult{}, err
		}
		found := false
		for i, v := range r.offers[c.SupplierID] {
			if v.ID != c.ResourceID {
				continue
			}
			found = true
			if !v.UpdatedAt.Equal(c.ExpectedUpdatedAt) {
				return domain.MasterDataMutationResult{}, domain.ErrConflict
			}
			m, _ := r.GetMaterial(ctx, v.MaterialID)
			if *c.Active && (s.Status != domain.SupplierActive || m.Status != domain.MaterialActive) {
				return domain.MasterDataMutationResult{}, domain.ErrInvalidState
			}
			v.Active = *c.Active
			v.UpdatedAt = now
			r.offers[s.ID][i] = v
			item = v
			if !v.Active && m.PreferredSupplierID != nil && *m.PreferredSupplierID == s.ID {
				m.PreferredSupplierID = nil
				m.UpdatedAt = now
				r.materials[m.ID] = m
			}
		}
		if !found {
			return domain.MasterDataMutationResult{}, domain.ErrNotFound
		}
	default:
		return domain.MasterDataMutationResult{}, domain.ErrInvalidArgument
	}
	snapshot, err := json.Marshal(item)
	if err != nil {
		return domain.MasterDataMutationResult{}, err
	}
	result := domain.MasterDataMutationResult{RequestHash: c.RequestHash, Snapshot: snapshot}
	if r.masterMutations == nil {
		r.masterMutations = map[string]domain.MasterDataMutationResult{}
	}
	r.masterMutations[c.Actor+"\x1f"+c.Key] = result
	return result, nil
}

type failedMasterMutationLookup struct{ *httpRepository }

func (r failedMasterMutationLookup) GetMasterDataMutationResult(context.Context, string, string) (domain.MasterDataMutationResult, error) {
	return domain.MasterDataMutationResult{}, errors.New("private master mutation database error")
}

func TestMasterDataMutationContractExamples(t *testing.T) {
	repo := newHTTPRepository()
	handler := newHandler(repo)
	now := time.Now().UTC()
	supplier, _ := domain.NewSupplier("EDIT-SUP", "Supplier", "COMPANY", "", "", "", "")
	supplier.CreatedAt, supplier.UpdatedAt = now, now
	material, _ := domain.NewRawMaterial("EDIT-RICE", "Rice", "KG", "")
	material.CreatedAt, material.UpdatedAt = now, now
	offer, _ := domain.NewSupplierMaterialOfferVersion(supplier.ID, material.ID, "", "KG", 12000, "VND", "1", 1, now.Add(-time.Hour), nil)
	offer.CreatedAt, offer.UpdatedAt = now, now
	repo.suppliers[supplier.ID] = supplier
	repo.materials[material.ID] = material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	prefix := "/api/v1/admin/procurement/"
	sp := prefix + "suppliers/" + supplier.ID.String()
	mp := prefix + "materials/" + material.ID.String()
	op := sp + "/offers/" + offer.ID.String() + "/status"
	check := func(method, path, body, role, key string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		if role != "" {
			r.Header.Set("Authorization", "Bearer "+employeeToken("master-editor", role))
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s got %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		var v map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && v["requestId"] != w.Header().Get("X-Request-Id") {
			t.Fatal("missing request correlation")
		}
		if strings.Contains(w.Body.String(), "private master") {
			t.Fatal("private error leaked")
		}
		fixture := map[string]any{"method": method, "path": path, "status": status, "body": v}
		var input any
		if json.Unmarshal([]byte(body), &input) == nil {
			fixture["request"] = input
		}
		encoded, _ := json.Marshal(fixture)
		fmt.Println("PROCUREMENT_HTTP=" + string(encoded))
		return v
	}
	type testCase struct {
		method, path, fields string
		stamp                func() time.Time
		id                   uuid.UUID
	}
	tests := []testCase{
		{"PUT", sp, `"code":"EDIT-SUP","name":"Edited supplier","supplier_type":"COMPANY"`, func() time.Time { return repo.suppliers[supplier.ID].UpdatedAt }, supplier.ID},
		{"PUT", mp, `"code":"EDIT-RICE","name":"Edited rice","base_unit":"KG","description":"Updated"`, func() time.Time { return repo.materials[material.ID].UpdatedAt }, material.ID},
		{"PATCH", sp + "/status", `"status":"ACTIVE"`, func() time.Time { return repo.suppliers[supplier.ID].UpdatedAt }, supplier.ID},
		{"PATCH", mp + "/status", `"status":"ACTIVE"`, func() time.Time { return repo.materials[material.ID].UpdatedAt }, material.ID},
		{"PUT", mp + "/preferred-supplier", `"supplier_id":"` + supplier.ID.String() + `"`, func() time.Time { return repo.materials[material.ID].UpdatedAt }, material.ID},
		{"PATCH", op, `"active":false`, func() time.Time { return repo.offers[supplier.ID][0].UpdatedAt }, offer.ID},
	}
	bodies := make([]string, len(tests))
	for i, tc := range tests {
		stamp := tc.stamp().Format(time.RFC3339Nano)
		body := fmt.Sprintf(`{%s,"expected_updated_at":%q}`, tc.fields, stamp)
		bodies[i] = body
		key := fmt.Sprintf("master-mutation-http-%04d", i)
		check(tc.method, tc.path, body, "", key, 401)
		for _, role := range []string{"VIEWER", "WAREHOUSE_STAFF", "UNASSIGNED"} {
			check(tc.method, tc.path, body, role, key, 403)
		}
		check(tc.method, tc.path, body, "PO_EDITOR", "", 400)
		for _, invalid := range []string{`{}`, `{`, fmt.Sprintf(`{%s,"expected_updated_at":"bad"}`, tc.fields), strings.TrimSuffix(body, "}") + `,"unknown":true}`} {
			check(tc.method, tc.path, invalid, "PO_EDITOR", key, 400)
		}
		missing := strings.Replace(tc.path, tc.id.String(), uuid.NewString(), 1)
		check(tc.method, missing, body, "PO_EDITOR", key+"-missing", 404)
		first := check(tc.method, tc.path, body, "PO_EDITOR", key, 200)
		if replay := check(tc.method, tc.path, body, "PO_EDITOR", key, 200); !reflect.DeepEqual(first, replay) {
			t.Fatal("replay changed committed result")
		}
		check(tc.method, tc.path, strings.Replace(body, stamp, tc.stamp().Add(time.Second).Format(time.RFC3339Nano), 1), "PO_EDITOR", key, 409)
		check(tc.method, tc.path, body, "PO_EDITOR", key+"-stale", 409)
	}
	if repo.materials[material.ID].PreferredSupplierID != nil {
		t.Fatal("offer deactivation did not clear preference")
	}
	// Replaying an earlier edit returns its frozen result without undoing later cleanup.
	check("PUT", mp, bodies[1], "PO_EDITOR", "master-mutation-http-0001", 200)
	check("PUT", mp+"/preferred-supplier", bodies[4], "PO_EDITOR", "master-mutation-http-0004", 200)
	changed := fmt.Sprintf(`{"code":"NEW-CODE","name":"Rice","base_unit":"KG","expected_updated_at":%q}`, repo.materials[material.ID].UpdatedAt.Format(time.RFC3339Nano))
	check("PUT", mp, changed, "PO_EDITOR", "master-code-immutable-0001", 409)
	changed = strings.Replace(changed, `"NEW-CODE"`, `"EDIT-RICE"`, 1)
	changed = strings.Replace(changed, `"KG"`, `"G"`, 1)
	check("PUT", mp, changed, "PO_EDITOR", "master-unit-immutable-0001", 409)
	var err error
	handler, err = httpapi.NewRouter(application.NewService(failedMasterMutationLookup{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	for i, tc := range tests {
		check(tc.method, tc.path, bodies[i], "PO_EDITOR", fmt.Sprintf("master-mutation-failure-%04d", i), 500)
	}
}
