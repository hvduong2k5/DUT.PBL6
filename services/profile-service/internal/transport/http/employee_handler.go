package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.EmployeeCode == "" || req.FullName == "" || req.IDCardNumber == "" || req.DepartmentID == "" {
		writeJSONError(w, http.StatusBadRequest, "employee_code, full_name, id_card_number and department_id are required")
		return
	}

	emp, err := h.employeeUsecase.CreateEmployee(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

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

	detail, err := h.employeeUsecase.GetEmployeeDetail(r.Context(), empID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
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

	if limit <= 0 {
		limit = 20
	}

	employees, err := h.employeeUsecase.ListEmployees(r.Context(), deptID, status, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, employees)
}
