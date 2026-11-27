package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

type readContractRepository struct {
	*httpRepository
	fail bool
}

func (r *readContractRepository) ListSuppliers(_ context.Context, limit, offset int) ([]domain.Supplier, error) {
	if r.fail {
		return nil, errors.New("private database connection detail")
	}
	// One-record fixture; nil intentionally simulates an empty repository result.
	if offset > 0 {
		return nil, nil
	}
	for _, item := range r.suppliers {
		return []domain.Supplier{item}, nil
	}
	return nil, nil
}

func (r *readContractRepository) ListMaterials(_ context.Context, limit, offset int) ([]domain.RawMaterial, error) {
	if r.fail {
		return nil, errors.New("private database connection detail")
	}
	if offset > 0 {
		return nil, nil
	}
	for _, item := range r.materials {
		return []domain.RawMaterial{item}, nil
	}
	return nil, nil
}

func TestAdminReadContractExamples(t *testing.T) {
	repo := &readContractRepository{httpRepository: newHTTPRepository()}
	supplier, _ := domain.NewSupplierWithLegalIdentifier("SUP-01", "Demo supplier", "COMPANY", "Contact", "0900000001", "supplier@example.test", "Demo address", "TAX-01")
	material, _ := domain.NewRawMaterial("RICE", "Rice", "KG", "Raw material")
	now := time.Now().UTC()
	supplier.CreatedAt, supplier.UpdatedAt = now, now
	material.CreatedAt, material.UpdatedAt = now, now
	material.PreferredSupplierID = &supplier.ID
	offer, _ := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "SUP-RICE", "KG", 25000, "VND", "1.5", 2)
	end := now.Add(24 * time.Hour)
	offer.EffectiveTo, offer.CreatedAt, offer.UpdatedAt = &end, now, now
	repo.suppliers[supplier.ID], repo.materials[material.ID] = supplier, material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	handler, err := httpapi.NewRouter(application.NewService(repo), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/admin/procurement/"
	paths := []string{
		base + "suppliers", base + "suppliers/" + supplier.ID.String(), base + "suppliers/" + supplier.ID.String() + "/offers",
		base + "materials", base + "materials/" + material.ID.String(), base + "materials/" + material.ID.String() + "/offers",
	}
	check := func(method, path, role string, expected int) {
		t.Helper()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("X-Request-Id", "gateway-test#1")
		request.Header.Set("X-User-Role", "ADMIN")
		request.Header.Set("X-Internal-Token", "not-user-auth")
		if role != "" {
			request.Header.Set("Authorization", "Bearer "+employeeToken("contract-user", role))
		}
		handler.ServeHTTP(recorder, request)
		if recorder.Code != expected {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, recorder.Code, expected, recorder.Body.String())
		}
		if recorder.Header().Get("Content-Type") != "application/json" || recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("missing JSON/no-store headers: %v", recorder.Header())
		}
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if expected >= 400 {
			if len(body) != 3 || body["requestId"] != "gateway-test#1" || body["requestId"] != recorder.Header().Get("X-Request-Id") || body["code"] == "" {
				t.Fatalf("not the shared Error envelope: %v", body)
			}
			if strings.Contains(recorder.Body.String(), "private database") {
				t.Fatal("dependency error leaked into response")
			}
		}
		fixture, _ := json.Marshal(map[string]any{"method": method, "path": path, "status": expected, "body": body})
		fmt.Println("PROCUREMENT_HTTP=" + string(fixture))
	}
	for _, path := range paths {
		check("GET", path, "PROCUREMENT_VIEWER", 200)
		check("GET", path, "", 401)
		check("GET", path, "UNASSIGNED", 403)
		check("GET", path, "WAREHOUSE_STAFF", 403)
		check("GET", path, "WAREHOUSE_VIEWER", 403)
	}
	for _, family := range []string{"suppliers", "materials"} {
		check("GET", base+family+"?limit=1&offset=1", "ADMIN", 200)
		for _, query := range []string{"limit=0", "limit=101", "offset=-1", "limit=abc"} {
			check("GET", base+family+"?"+query, "ADMIN", 400)
		}
		for _, suffix := range []string{"", "/offers"} {
			check("GET", base+family+"/bad-id"+suffix, "ADMIN", 400)
			check("GET", base+family+"/"+uuid.NewString()+suffix, "ADMIN", 404)
		}
	}
	delete(repo.offers, supplier.ID)
	check("GET", paths[2], "ADMIN", 200)
	check("GET", paths[5], "ADMIN", 200)
	repo.fail = true
	check("GET", paths[0], "ADMIN", 500)
	check("GET", paths[3], "ADMIN", 500)
	check("GET", base+"unknown", "ADMIN", 404)
	check("DELETE", paths[0], "ADMIN", 405)
}

func TestInvalidCorrelationIDIsReplaced(t *testing.T) {
	handler := newHandler(newHTTPRepository())
	for _, value := range []string{"<untrusted>", strings.Repeat("a", 129), "", "space here"} {
		request := httptest.NewRequest("GET", "/api/v1/admin/procurement/suppliers", nil)
		request.Header.Set("X-Request-Id", value)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var body map[string]any
		_ = json.Unmarshal(recorder.Body.Bytes(), &body)
		id := recorder.Header().Get("X-Request-Id")
		if _, err := uuid.Parse(id); err != nil || body["requestId"] != id {
			t.Fatalf("invalid generated correlation ID: %v", body)
		}
	}
}
