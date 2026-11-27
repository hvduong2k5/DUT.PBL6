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

func (r *httpRepository) GetMasterDataCreateResult(_ context.Context, actor, key string) (domain.MasterDataCreateResult, error) {
	result, ok := r.masterResults[actor+"\x1f"+key]
	if !ok {
		return result, domain.ErrNotFound
	}
	return result, nil
}
func (r *httpRepository) CreateMasterDataCommand(ctx context.Context, cmd domain.MasterDataCreateCommand) (domain.MasterDataCreateResult, error) {
	var item any
	now := time.Now().UTC()
	switch cmd.Kind {
	case "SUPPLIER":
		v := *cmd.Supplier
		for _, old := range r.suppliers {
			if old.Code == v.Code || (v.LegalIdentifier != "" && old.LegalIdentifier == v.LegalIdentifier) {
				return domain.MasterDataCreateResult{}, domain.ErrConflict
			}
		}
		v.CreatedAt, v.UpdatedAt = now, now
		r.suppliers[v.ID] = v
		item = v
	case "MATERIAL":
		v := *cmd.Material
		for _, old := range r.materials {
			if old.Code == v.Code {
				return domain.MasterDataCreateResult{}, domain.ErrConflict
			}
		}
		v.CreatedAt, v.UpdatedAt = now, now
		r.materials[v.ID] = v
		item = v
	case "OFFER":
		v := *cmd.Offer
		s, err := r.GetSupplier(ctx, v.SupplierID)
		if err != nil {
			return domain.MasterDataCreateResult{}, err
		}
		m, err := r.GetMaterial(ctx, v.MaterialID)
		if err != nil {
			return domain.MasterDataCreateResult{}, err
		}
		if s.Status != domain.SupplierActive || m.Status != domain.MaterialActive || v.Unit != m.BaseUnit {
			return domain.MasterDataCreateResult{}, domain.ErrInvalidState
		}
		for _, old := range r.offers[v.SupplierID] {
			if old.MaterialID == v.MaterialID && old.EffectiveFrom.Equal(v.EffectiveFrom) {
				return domain.MasterDataCreateResult{}, domain.ErrConflict
			}
		}
		v.CreatedAt, v.UpdatedAt = now, now
		r.offers[v.SupplierID] = append(r.offers[v.SupplierID], v)
		item = v
	default:
		return domain.MasterDataCreateResult{}, domain.ErrInvalidArgument
	}
	snapshot, err := json.Marshal(item)
	if err != nil {
		return domain.MasterDataCreateResult{}, err
	}
	result := domain.MasterDataCreateResult{Kind: cmd.Kind, RequestHash: cmd.RequestHash, Snapshot: snapshot}
	if r.masterResults == nil {
		r.masterResults = map[string]domain.MasterDataCreateResult{}
	}
	r.masterResults[cmd.Actor+"\x1f"+cmd.Key] = result
	return result, nil
}

type failedMasterLookup struct{ *httpRepository }

func (r failedMasterLookup) GetMasterDataCreateResult(context.Context, string, string) (domain.MasterDataCreateResult, error) {
	return domain.MasterDataCreateResult{}, errors.New("private database connection failure")
}

func TestMasterDataCreateRejectsInvalidApplicationInputs(t *testing.T) {
	service := application.NewService(nil) // Validation must not consult persistence.
	supplier, _ := domain.NewSupplier("VALID", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("VALID", "Rice", "KG", "")
	for _, cmd := range []domain.MasterDataCreateCommand{
		{},
		{Kind: "UNKNOWN", Actor: "staff", Key: "master-invalid-0001"},
		{Kind: "SUPPLIER", Actor: "staff", Key: "master-invalid-0001"},
		{Kind: "SUPPLIER", Actor: "staff", Key: "master-invalid-0001", Supplier: &supplier, Material: &material},
		{Kind: "MATERIAL", Actor: "staff", Key: "master-invalid-0001"},
		{Kind: "MATERIAL", Actor: "staff", Key: "master-invalid-0001", Material: &domain.RawMaterial{}},
		{Kind: "SUPPLIER", Actor: "staff", Key: "master-invalid-0001", Supplier: &domain.Supplier{}},
		{Kind: "SUPPLIER", Actor: "", Key: "master-invalid-0001", Supplier: &supplier},
		{Kind: "SUPPLIER", Actor: "staff", Key: "short", Supplier: &supplier},
		{Kind: "OFFER", Actor: "staff", Key: "master-invalid-0001"},
		{Kind: "OFFER", Actor: "staff", Key: "master-invalid-0001", Offer: &domain.SupplierMaterialOffer{}},
	} {
		if _, err := service.CreateMasterDataWithIdempotency(context.Background(), cmd); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("%s accepted invalid input: %v", cmd.Kind, err)
		}
	}
}

func TestMasterDataCreateContractExamples(t *testing.T) {
	repo := newHTTPRepository()
	handler := newHandler(repo)
	check := func(path, body, role, key string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		if role != "" {
			r.Header.Set("Authorization", "Bearer "+employeeToken("master-operator", role))
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s got %d want %d: %s", path, w.Code, status, w.Body.String())
		}
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && value["requestId"] != w.Header().Get("X-Request-Id") {
			t.Fatal("missing correlation")
		}
		if strings.Contains(w.Body.String(), "private database") {
			t.Fatal("private error leaked")
		}
		fixture := map[string]any{"method": "POST", "path": path, "status": status, "body": value}
		var input any
		if json.Unmarshal([]byte(body), &input) == nil {
			fixture["request"] = input
		}
		encoded, _ := json.Marshal(fixture)
		fmt.Println("PROCUREMENT_HTTP=" + string(encoded))
		return value
	}
	prefix := "/api/v1/admin/procurement/"
	supplierBody := `{"code":"MASTER-SUP","name":"Supplier","supplier_type":"COMPANY","legal_identifier":" TAX-01 "}`
	materialBody := `{"code":"MASTER-RICE","name":"Rice","base_unit":"KG","description":"Food"}`
	supplier := check(prefix+"suppliers", supplierBody, "PO_EDITOR", "master-supplier-0001", 201)
	material := check(prefix+"materials", materialBody, "PO_EDITOR", "master-material-0001", 201)
	offerPath := prefix + "suppliers/" + supplier["id"].(string) + "/offers"
	offerBody := fmt.Sprintf(`{"material_id":%q,"unit":"KG","unit_price_minor":12000,"currency":"VND","minimum_quantity":"1","lead_time_days":2,"effective_from":"2026-11-28","effective_to":"2026-12-01"}`, material["id"])
	for _, test := range []struct {
		path, body, key string
		first           map[string]any
	}{
		{prefix + "suppliers", supplierBody, "master-supplier-0001", supplier},
		{prefix + "materials", materialBody, "master-material-0001", material},
		{offerPath, offerBody, "master-offer-key-0001", nil},
	} {
		check(test.path, test.body, "", test.key, 401)
		for _, role := range []string{"VIEWER", "WAREHOUSE_STAFF", "UNASSIGNED"} {
			check(test.path, test.body, role, test.key, 403)
		}
		for _, key := range []string{"", "short", strings.Repeat("x", 129), "invalid key with spaces"} {
			check(test.path, test.body, "PO_EDITOR", key, 400)
		}
		for _, body := range []string{`{}`, `{`, strings.TrimSuffix(test.body, "}") + `,"unknown":true}`} {
			check(test.path, body, "PO_EDITOR", test.key, 400)
		}
		first := test.first
		if first == nil {
			first = check(test.path, test.body, "PO_EDITOR", test.key, 201)
		}
		if replay := check(test.path, test.body, "PO_EDITOR", test.key, 201); !reflect.DeepEqual(first, replay) {
			t.Fatal("retry did not return frozen result")
		}
		changed := test.body
		if test.path == offerPath {
			changed = strings.Replace(test.body, "12000", "13000", 1)
		} else {
			changed = strings.Replace(test.body, `"name":"`, `"name":"Changed `, 1)
		}
		check(test.path, changed, "PO_EDITOR", test.key, 409)
		check(test.path, test.body, "PO_EDITOR", test.key+"-duplicate", 409)
	}
	check(prefix+"materials", materialBody, "PO_EDITOR", "master-supplier-0001", 409)
	check(prefix+"suppliers", strings.Replace(supplierBody, "MASTER-SUP", strings.Repeat("x", 81), 1), "PO_EDITOR", "master-large-supplier-0001", 400)
	check(prefix+"materials", strings.Replace(materialBody, "MASTER-RICE", strings.Repeat("x", 81), 1), "PO_EDITOR", "master-large-material-0001", 400)
	check(offerPath, strings.Replace(offerBody, `"minimum_quantity":"1"`, `"minimum_quantity":"1000000000000"`, 1), "PO_EDITOR", "master-large-quantity-0001", 400)
	check(offerPath, strings.Replace(offerBody, `"lead_time_days":2`, `"lead_time_days":2147483648`, 1), "PO_EDITOR", "master-large-lead-time-0001", 400)
	check(prefix+"suppliers/"+uuid.NewString()+"/offers", offerBody, "PO_EDITOR", "master-missing-parent-0001", 404)
	for _, body := range []string{strings.Replace(offerBody, `"2026-11-28"`, `""`, 1), strings.Replace(offerBody, `"2026-11-28"`, `"bad-date"`, 1), strings.Replace(offerBody, `"2026-12-01"`, `"2026-11-26"`, 1), strings.Replace(offerBody, `"material_id":"`, `"material_id":"bad`, 1)} {
		check(offerPath, body, "PO_EDITOR", "master-invalid-offer-0001", 400)
	}
	for _, offers := range repo.offers {
		if offers[0].EffectiveTo.Format("2006-01-02") != "2026-12-02" {
			t.Fatal("inclusive end date was not converted to exclusive UTC boundary")
		}
	}
	sid := uuid.MustParse(supplier["id"].(string))
	s := repo.suppliers[sid]
	s.Status = domain.SupplierSuspended
	repo.suppliers[sid] = s
	if replay := check(prefix+"suppliers", strings.Replace(supplierBody, " TAX-01 ", "tax-01", 1), "PO_EDITOR", "master-supplier-0001", 201); !reflect.DeepEqual(supplier, replay) {
		t.Fatal("normalization or frozen snapshot failed")
	}
	check(offerPath, offerBody, "PO_EDITOR", "master-offer-key-0001", 201)
	check(offerPath, strings.Replace(offerBody, "2026-11-28", "2026-11-29", 1), "PO_EDITOR", "master-inactive-parent-0001", 409)
	var err error
	handler, err = httpapi.NewRouter(application.NewService(failedMasterLookup{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ path, body string }{{prefix + "suppliers", supplierBody}, {prefix + "materials", materialBody}, {offerPath, offerBody}} {
		check(test.path, test.body, "PO_EDITOR", "master-db-failure-0001", 500)
	}
}
