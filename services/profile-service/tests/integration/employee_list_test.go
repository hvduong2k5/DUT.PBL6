package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/security"
	"github.com/omamx/profile-service/internal/repository"
	profilehttp "github.com/omamx/profile-service/internal/transport/http"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/omamx/profile-service/internal/usecase"
	"github.com/omamx/profile-service/tests/testhelper"
	"github.com/stretchr/testify/require"
)

func TestEmployeeListReturnsJSONArrays(t *testing.T) {
	if testing.Short() {
		t.Skip("real PostgreSQL integration")
	}
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()
	repo := repository.NewEmployeeRepository(pool)
	handler := profilehttp.NewEmployeeHandler(usecase.NewEmployeeUsecase(repo, nil))
	actor := security.Principal{
		Role:   "EMPLOYEE",
		Grants: []domain.PermissionGrant{{PermissionCode: "EMPLOYEE_PROFILE_VIEW", ScopeType: "GLOBAL"}},
	}
	list := func() *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/employees", nil)
		handler.ListEmployees(response, request.WithContext(context.WithValue(request.Context(), middleware.PrincipalKey, actor)))
		require.Equal(t, http.StatusOK, response.Code)
		return response
	}
	require.JSONEq(t, "[]", list().Body.String())

	now := time.Now().UTC()
	require.NoError(t, repo.CreateEmployee(context.Background(), &domain.EmployeeProfile{
		ID: uuid.New(), EmployeeCode: "HR-001", FullName: "HR Test",
		DepartmentID: "DEPT-WAREHOUSE", Position: "Staff", ContractType: "FULLTIME",
		Status: "ACTIVE", ContractStartDate: now, CreatedAt: now, UpdatedAt: now,
		IDCardEncrypted: []byte{}, IDCardNonce: []byte{}, EncryptedDEK: []byte{}, KEKVersion: 1,
	}))
	require.Contains(t, list().Body.String(), `"employee_code":"HR-001"`)
}
