package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/omamx/profile-service/internal/usecase"
)

type EmployeeHandler struct {
	employeeUsecase *usecase.EmployeeUsecase
}

func NewEmployeeHandler(employeeUsecase *usecase.EmployeeUsecase) *EmployeeHandler {
	return &EmployeeHandler{
		employeeUsecase: employeeUsecase,
	}
}

// CreateEmployee handles POST /api/v1/admin/employees
func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateEmployeeRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.EmployeeCode == "" || req.FullName == "" || req.IDCardNumber == "" || req.DepartmentID == "" {
		writeJSONError(w, http.StatusBadRequest, "employee_code, full_name, id_card_number and department_id are required")
		return
	}
	if !middleware.StaffScope(r.Context(), "EMPLOYEE_PROFILE_CREATE").Allows(req.DepartmentID) {
		writeJSON(w, 403, map[string]string{"code": "PERMISSION_DENIED"})
		return
	}

	emp, err := h.employeeUsecase.CreateEmployee(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	maskEmployee(r, emp)
	writeJSON(w, http.StatusCreated, emp)
}

// GetEmployeeDetail handles GET /api/v1/admin/employees/{id}
func (h *EmployeeHandler) GetEmployeeDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	empID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid employee id format")
		return
	}

	detail, err := h.employeeUsecase.GetEmployeeDetailScoped(r.Context(), empID,
		middleware.StaffScope(r.Context(), "EMPLOYEE_PROFILE_VIEW"), middleware.StaffScope(r.Context(), "EMPLOYEE_PROFILE_SENSITIVE_VIEW"))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// ListEmployees handles GET /api/v1/admin/employees
func (h *EmployeeHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	deptID := r.URL.Query().Get("department_id")
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if offset < 0 {
		writeJSONError(w, 400, "offset must be non-negative")
		return
	}
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 20
	}

	scope := middleware.StaffScope(r.Context(), "EMPLOYEE_PROFILE_VIEW")
	if deptID != "" && !scope.Allows(deptID) {
		writeJSON(w, 403, map[string]string{"code": "PERMISSION_DENIED"})
		return
	}
	employees, err := h.employeeUsecase.ListEmployeesScoped(r.Context(), deptID, status, limit, offset, scope)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	for _, emp := range employees {
		maskEmployee(r, emp)
	}
	writeJSON(w, http.StatusOK, employees)
}

// GetCompliance handles GET /api/v1/profile/employees/compliance and /api/v1/admin/employees/compliance
func (h *EmployeeHandler) GetCompliance(w http.ResponseWriter, r *http.Request) {
	thresholdDays := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		d, err := strconv.Atoi(daysStr)
		if err != nil || d < 1 || d > 365 {
			writeDomainError(w, &domain.ValidationError{Field: "days", Message: "must be between 1 and 365"})
			return
		}
		thresholdDays = d
	}

	expiring, err := h.employeeUsecase.FindExpiringCertificatesScoped(r.Context(), thresholdDays, middleware.StaffScope(r.Context(), "EMPLOYEE_COMPLIANCE_VIEW"))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	if expiring == nil {
		expiring = []*domain.EmployeeProfile{}
	}
	for _, emp := range expiring {
		maskEmployee(r, emp)
	}
	writeJSON(w, http.StatusOK, expiring)
}

func maskEmployee(r *http.Request, emp *domain.EmployeeProfile) {
	if !middleware.StaffScope(r.Context(), "EMPLOYEE_PROFILE_SENSITIVE_VIEW").Allows(emp.DepartmentID) {
		emp.PhoneNumber = ""
		emp.FoodSafetyCertNo = ""
	}
}
