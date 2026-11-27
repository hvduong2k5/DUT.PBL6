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

func (r *httpRepository) GetSupplierContractCommandResult(_ context.Context, actor, key string) (domain.SupplierContractCommandResult, error) {
	v, ok := r.contractCommands[actor+"\x1f"+key]
	if !ok {
		return v, domain.ErrNotFound
	}
	return v, nil
}
func (r *httpRepository) ExecuteSupplierContractCommand(ctx context.Context, c domain.SupplierContractCommand) (domain.SupplierContractCommandResult, error) {
	var v domain.SupplierContract
	var err error
	if c.Action == "CREATE" {
		v = *c.Contract
		s, e := r.GetSupplier(ctx, v.SupplierID)
		if e != nil {
			return domain.SupplierContractCommandResult{}, e
		}
		if s.Status != domain.SupplierActive {
			return domain.SupplierContractCommandResult{}, domain.ErrInvalidState
		}
		v.Revision = 1
		for _, old := range r.contracts {
			if old.SupplierID == v.SupplierID && old.ContractNumber == v.ContractNumber && old.Revision >= v.Revision {
				v.Revision = old.Revision + 1
			}
		}
		v.CreatedAt = time.Now().UTC()
		v.UpdatedAt = v.CreatedAt
	} else {
		v, err = r.GetSupplierContract(ctx, c.ContractID)
		if err != nil {
			return domain.SupplierContractCommandResult{}, err
		}
		if !v.UpdatedAt.Equal(c.ExpectedUpdatedAt) {
			return domain.SupplierContractCommandResult{}, domain.ErrConflict
		}
		if c.Action == "ACTIVATE" {
			s, e := r.GetSupplier(ctx, v.SupplierID)
			if e != nil {
				return domain.SupplierContractCommandResult{}, e
			}
			if s.Status != domain.SupplierActive || v.Status != domain.SupplierContractDraft || v.EffectiveTo != nil && !time.Now().UTC().Before(*v.EffectiveTo) {
				return domain.SupplierContractCommandResult{}, domain.ErrInvalidState
			}
			for _, old := range r.contracts {
				if old.ID != v.ID && old.SupplierID == v.SupplierID && strings.EqualFold(old.Scope, v.Scope) && old.Status == domain.SupplierContractActive && (v.EffectiveTo == nil || old.EffectiveFrom.Before(*v.EffectiveTo)) && (old.EffectiveTo == nil || v.EffectiveFrom.Before(*old.EffectiveTo)) {
					return domain.SupplierContractCommandResult{}, domain.ErrConflict
				}
			}
			v.Status = domain.SupplierContractActive
		} else {
			if v.Status != domain.SupplierContractActive {
				return domain.SupplierContractCommandResult{}, domain.ErrInvalidState
			}
			v.Status = domain.SupplierContractTerminated
		}
		v.UpdatedAt = time.Now().UTC()
	}
	r.contracts[v.ID] = v
	snapshot, e := json.Marshal(v)
	if e != nil {
		return domain.SupplierContractCommandResult{}, e
	}
	result := domain.SupplierContractCommandResult{RequestHash: c.RequestHash, Snapshot: snapshot}
	if r.contractCommands == nil {
		r.contractCommands = map[string]domain.SupplierContractCommandResult{}
	}
	r.contractCommands[c.Actor+"\x1f"+c.Key] = result
	return result, nil
}

type failedContractRepository struct{ *httpRepository }

func (r failedContractRepository) GetSupplierContractCommandResult(context.Context, string, string) (domain.SupplierContractCommandResult, error) {
	return domain.SupplierContractCommandResult{}, errors.New("private contract database error")
}
func (r failedContractRepository) GetSupplierContract(context.Context, uuid.UUID) (domain.SupplierContract, error) {
	return domain.SupplierContract{}, errors.New("private contract database error")
}
func (r failedContractRepository) ListSupplierContracts(context.Context, uuid.UUID, int, int) ([]domain.SupplierContract, error) {
	return nil, errors.New("private contract database error")
}

func TestSupplierContractCommandRejectsInvalidInputBeforePersistence(t *testing.T) {
	service := application.NewService(nil)
	base := domain.SupplierContractCommand{Action: "CREATE", Actor: "staff", Key: "contract-invalid-0001"}
	contract, _ := domain.NewSupplierContract(uuid.New(), "AGR", "RAW", "NET 30", "", time.Now().UTC(), nil, "staff")
	unverified := contract
	unverified.DocumentFileID = "https://unverified.invalid/document"
	for _, c := range []domain.SupplierContractCommand{
		{}, base,
		{Action: "CREATE", Actor: "staff", Key: "short", Contract: &contract},
		{Action: "CREATE", Actor: "", Key: base.Key, Contract: &contract},
		{Action: "CREATE", Actor: "staff", Key: base.Key, Contract: &unverified},
		{Action: "CREATE", Actor: "staff", Key: base.Key, Contract: &contract, ContractID: uuid.New()},
		{Action: "ACTIVATE", Actor: "staff", Key: base.Key, ContractID: uuid.New()},
		{Action: "ACTIVATE", Actor: "staff", Key: base.Key, ContractID: uuid.New(), ExpectedUpdatedAt: time.Now(), Reason: "Not accepted"},
		{Action: "TERMINATE", Actor: "staff", Key: base.Key, ContractID: uuid.New(), ExpectedUpdatedAt: time.Now(), Reason: " "},
		{Action: "TERMINATE", Actor: "staff", Key: base.Key, ContractID: uuid.New(), ExpectedUpdatedAt: time.Now(), Reason: strings.Repeat("x", 2001)},
	} {
		if _, err := service.ExecuteSupplierContractCommand(context.Background(), c); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("invalid %s reached persistence: %v", c.Action, err)
		}
	}
}

func TestSupplierContractContractExamples(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("CONTRACT-SUP", "Supplier", "COMPANY", "", "", "", "")
	repo.suppliers[supplier.ID] = supplier
	handler := newHandler(repo)
	prefix := "/api/v1/admin/procurement/"
	listPath := prefix + "suppliers/" + supplier.ID.String() + "/contracts"
	check := func(method, path, body, role, key string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		if role != "" {
			r.Header.Set("Authorization", "Bearer "+employeeToken("contract-operator", role))
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
			t.Fatal("missing correlation")
		}
		if strings.Contains(w.Body.String(), "private contract") {
			t.Fatal("private database detail leaked")
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
	check("GET", listPath, "", "", "", 401)
	for _, role := range []string{"VIEWER", "WAREHOUSE_STAFF"} {
		check("GET", listPath, "", role, "", 403)
	}
	empty := check("GET", listPath, "", "PO_EDITOR", "", 200)
	if len(empty["items"].([]any)) != 0 {
		t.Fatal("expected empty page")
	}
	check("GET", listPath+"?limit=0", "", "PO_EDITOR", "", 400)
	check("GET", prefix+"suppliers/"+uuid.NewString()+"/contracts", "", "PO_EDITOR", "", 404)
	createBody := `{"contract_number":"AGR-NEW","scope":" raw_materials ","payment_terms":"NET 30","effective_from":"2026-11-01"}`
	createKey := "contract-http-create-0001"
	check("POST", listPath, createBody, "", createKey, 401)
	for _, role := range []string{"VIEWER", "WAREHOUSE_STAFF"} {
		check("POST", listPath, createBody, role, createKey, 403)
	}
	check("POST", listPath, createBody, "PO_EDITOR", "", 400)
	for _, bad := range []string{`{}`, `{`, strings.Replace(createBody, "2026-11-01", "bad", 1), strings.TrimSuffix(createBody, "}") + `,"unknown":true}`, strings.TrimSuffix(createBody, "}") + `,"document_file_id":"https://example.invalid/document"}`, strings.TrimSuffix(createBody, "}") + `,"document_file_id":"unverified-file-id"}`, strings.Replace(createBody, "AGR-NEW", strings.Repeat("x", 101), 1)} {
		check("POST", listPath, bad, "PO_EDITOR", createKey, 400)
	}
	check("POST", prefix+"suppliers/"+uuid.NewString()+"/contracts", createBody, "PO_EDITOR", createKey, 404)
	created := check("POST", listPath, createBody, "PO_EDITOR", createKey, 201)
	id := created["id"].(string)
	detail := prefix + "contracts/" + id
	if replay := check("POST", listPath, strings.Replace(createBody, " raw_materials ", "RAW_MATERIALS", 1), "PO_EDITOR", createKey, 201); !reflect.DeepEqual(created, replay) {
		t.Fatal("normalized create replay changed")
	}
	check("POST", listPath, strings.Replace(createBody, "NET 30", "NET 60", 1), "PO_EDITOR", createKey, 409)
	check("GET", detail, "", "", "", 401)
	for _, role := range []string{"VIEWER", "WAREHOUSE_STAFF"} {
		check("GET", detail, "", role, "", 403)
	}
	check("GET", detail, "", "PO_EDITOR", "", 200)
	check("GET", prefix+"contracts/bad-id", "", "PO_EDITOR", "", 400)
	check("GET", prefix+"contracts/"+uuid.NewString(), "", "PO_EDITOR", "", 404)
	check("GET", listPath, "", "PO_EDITOR", "", 200)
	actionBody := func(stamp string, reason bool) string {
		if reason {
			return fmt.Sprintf(`{"expected_updated_at":%q,"reason":"Supplier agreement ended"}`, stamp)
		}
		return fmt.Sprintf(`{"expected_updated_at":%q}`, stamp)
	}
	for i, action := range []string{"activate", "terminate"} {
		current := repo.contracts[uuid.MustParse(id)]
		stamp := current.UpdatedAt.Format(time.RFC3339Nano)
		body := actionBody(stamp, action == "terminate")
		path := detail + "/" + action
		key := fmt.Sprintf("contract-http-action-%04d", i)
		check("POST", path, body, "", key, 401)
		for _, role := range []string{"VIEWER", "WAREHOUSE_STAFF"} {
			check("POST", path, body, role, key, 403)
		}
		check("POST", path, body, "PO_EDITOR", "", 400)
		for _, bad := range []string{`{}`, `{`, strings.Replace(body, stamp, "bad-time", 1), strings.TrimSuffix(body, "}") + `,"unknown":true}`} {
			check("POST", path, bad, "PO_EDITOR", key, 400)
		}
		if action == "terminate" {
			check("POST", path, actionBody(stamp, false), "PO_EDITOR", key, 400)
		} else {
			check("POST", path, actionBody(stamp, true), "PO_EDITOR", key, 400)
		}
		check("POST", prefix+"contracts/"+uuid.NewString()+"/"+action, body, "PO_EDITOR", key, 404)
		check("POST", path, body, "PO_EDITOR", createKey, 409)
		first := check("POST", path, body, "PO_EDITOR", key, 200)
		if replay := check("POST", path, body, "PO_EDITOR", key, 200); !reflect.DeepEqual(first, replay) {
			t.Fatal("action replay changed")
		}
		check("POST", path, body, "PO_EDITOR", key+"-stale", 409)
		check("POST", path, strings.Replace(body, stamp, time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano), 1), "PO_EDITOR", key, 409)
	}
	if replay := check("POST", listPath, createBody, "PO_EDITOR", createKey, 201); !reflect.DeepEqual(created, replay) {
		t.Fatal("create snapshot changed after termination")
	}
	revision := check("POST", listPath, createBody, "PO_EDITOR", createKey+"-revision", 201)
	if revision["revision"] != float64(2) {
		t.Fatal("new key did not create new immutable revision")
	}
	var err error
	handler, err = httpapi.NewRouter(application.NewService(failedContractRepository{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check("GET", listPath, "", "PO_EDITOR", "", 500)
	check("GET", detail, "", "PO_EDITOR", "", 500)
	check("POST", listPath, createBody, "PO_EDITOR", createKey, 500)
	for _, action := range []string{"activate", "terminate"} {
		check("POST", detail+"/"+action, actionBody(created["updated_at"].(string), action == "terminate"), "PO_EDITOR", "contract-failure-key-0001", 500)
	}
}
