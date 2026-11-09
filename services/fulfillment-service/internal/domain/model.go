package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const Contract = "order-3.2"

type Error struct {
	Code      string
	HTTP      int
	Retryable bool
}

func (e *Error) Error() string { return e.Code }
func Fail(code string, status int) error {
	return &Error{Code: code, HTTP: status, Retryable: status >= 500}
}
func UUID(s string) bool {
	id, e := uuid.Parse(s)
	return e == nil && id != uuid.Nil && id.String() == s
}

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var sealPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{5,79}$`)

func Key(s string) bool {
	if len(s) < 16 || len(s) > 128 {
		return false
	}
	for _, c := range []byte(s) {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}
func Hash(v any) string {
	raw, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	var normalized any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if e = decoder.Decode(&normalized); e != nil {
		panic(e)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(normalized)
	h := sha256.Sum256(bytes.TrimSuffix(b.Bytes(), []byte("\n")))
	return hex.EncodeToString(h[:])
}
func Digest(lines []Line) (string, error) {
	totals := map[string]int64{}
	for _, l := range lines {
		if l.Quantity <= 0 || l.SKU == "" {
			return "", Fail("INVALID_QUANTITY", 400)
		}
		for _, c := range []byte(l.SKU) {
			if c < 33 || c > 126 {
				return "", Fail("INVALID_SKU", 400)
			}
		}
		totals[l.SKU] += int64(l.Quantity)
		if totals[l.SKU] > math.MaxInt32 {
			return "", Fail("QUANTITY_OVERFLOW", 400)
		}
	}
	if len(totals) == 0 {
		return "", Fail("EMPTY_ITEMS", 400)
	}
	keys := make([]string, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	a := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		a = append(a, map[string]any{"quantity": totals[k], "sku_code": k})
	}
	return Hash(a), nil
}

type Principal struct {
	ID, Service             string
	Permissions, Warehouses []string
}

func (p Principal) Caller() string {
	if p.Service != "" {
		return "service:" + p.Service
	}
	return "staff:" + p.ID
}
func (p Principal) Can(permission, warehouse string) bool {
	if p.Service != "" || !UUID(p.ID) {
		return false
	}
	return contains(p.Permissions, permission) && contains(p.Warehouses, warehouse)
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

type Line struct {
	ID        string `json:"order_line_id"`
	SKU       string `json:"sku_code"`
	Name      string `json:"product_name"`
	Packaging string `json:"packaging_text"`
	Quantity  int32  `json:"quantity"`
	Fragile   bool   `json:"fragile"`
}
type Checklist struct {
	Code       string     `json:"item_code"`
	Definition string     `json:"definition"`
	Required   bool       `json:"required"`
	Result     string     `json:"result"`
	By         *string    `json:"confirmed_by"`
	At         *time.Time `json:"confirmed_at"`
	Note       *string    `json:"note"`
}
type Policy struct {
	Version         string      `json:"policy_version"`
	Evidence        string      `json:"evidence_mode"`
	Checklist       []Checklist `json:"checklist"`
	MaxWeight       int32       `json:"maximum_weight_grams"`
	MaxDimension    int32       `json:"maximum_dimension_cm"`
	MaxVideoBytes   int64       `json:"maximum_video_bytes"`
	MaxVideoSeconds int32       `json:"maximum_video_seconds"`
}
type Inputs struct {
	Schema           string     `json:"schema_version"`
	Contract         string     `json:"contract_version"`
	SnapshotID       string     `json:"snapshot_id"`
	Version          int64      `json:"snapshot_version"`
	OrderID          string     `json:"order_id"`
	Generation       int64      `json:"ready_generation"`
	Digest           string     `json:"items_digest"`
	Warehouse        string     `json:"warehouse_id"`
	ReservationID    string     `json:"reservation_id"`
	Channel          string     `json:"channel"`
	Due              *time.Time `json:"packing_due_at"`
	DeadlineRevision int64      `json:"deadline_revision"`
	Lines            []Line     `json:"lines"`
	Policy           Policy     `json:"policy"`
}
type Allocation struct {
	ID                 string     `json:"allocation_id"`
	LineID             string     `json:"order_line_id"`
	SKU                string     `json:"sku_code,omitempty"`
	BatchID            string     `json:"batch_id"`
	BatchCode          string     `json:"batch_code"`
	Allocated          int32      `json:"allocated_quantity"`
	Counted            int32      `json:"counted_quantity"`
	Version            int64      `json:"allocation_version"`
	Until              time.Time  `json:"batch_valid_until"`
	EligibilityVersion int64      `json:"eligibility_revision"`
	Eligible           bool       `json:"currently_eligible"`
	ValidatedAt        *time.Time `json:"last_validated_at"`
}
type Stock struct {
	Contract      string       `json:"contract_version"`
	OrderID       string       `json:"order_id"`
	ReservationID string       `json:"reservation_id"`
	State         string       `json:"state"`
	Version       int64        `json:"allocation_version"`
	Digest        string       `json:"items_digest"`
	Warehouse     string       `json:"warehouse_id"`
	Observed      time.Time    `json:"observed_at"`
	Slots         []Allocation `json:"slots"`
	DecisionID    string       `json:"original_commit_decision_id"`
	Generation    int64        `json:"ready_generation"`
}
type Validation struct {
	Allowed                                                     bool
	ID                                                          string
	OrderID, TaskID, ReservationID, Digest, Warehouse, Contract string
	Version, Generation                                         int64
	Observed, Until                                             time.Time
	Slots                                                       []Allocation
}

func (i Inputs) Validate(s Stock, now time.Time, integrated bool) error {
	d, e := Digest(i.Lines)
	if e != nil {
		return e
	}
	if i.Contract != Contract || s.Contract != Contract || i.Schema != "packing-inputs.v1" || !UUID(i.OrderID) || !UUID(i.ReservationID) || i.SnapshotID == "" || i.Version <= 0 || i.Generation <= 0 || i.Warehouse == "" || i.Channel == "" || i.Digest != d || i.Digest != s.Digest || i.OrderID != s.OrderID || i.Generation != s.Generation || i.Warehouse != s.Warehouse || i.ReservationID != s.ReservationID || s.State != "COMMITTED" || s.Version <= 0 || s.DecisionID == "" || s.Observed.After(now.Add(2*time.Second)) || s.Observed.IsZero() {
		return Fail("INPUT_PROOF_MISMATCH", 412)
	}
	if i.Policy.Version == "" || i.Policy.MaxWeight <= 0 || i.Policy.MaxDimension <= 0 || len(i.Policy.Checklist) == 0 || i.DeadlineRevision <= 0 || !contains([]string{"OPTIONAL", "VIDEO_REQUIRED"}, i.Policy.Evidence) || (integrated && i.Policy.Evidence != "VIDEO_REQUIRED") {
		return Fail("INVALID_POLICY", 412)
	}
	ids := map[string]Line{}
	checks := map[string]bool{}
	for _, l := range i.Lines {
		if !UUID(l.ID) || ids[l.ID].ID != "" {
			return Fail("INVALID_LINE", 412)
		}
		ids[l.ID] = l
	}
	for _, c := range i.Policy.Checklist {
		if c.Code == "" || checks[c.Code] {
			return Fail("INVALID_CHECKLIST", 412)
		}
		checks[c.Code] = true
	}
	sums := map[string]int64{}
	slots := map[string]bool{}
	batches := map[string]bool{}
	for _, a := range s.Slots {
		l, ok := ids[a.LineID]
		pair := a.LineID + ":" + a.BatchID
		if !ok || !UUID(a.ID) || !UUID(a.BatchID) || slots[a.ID] || batches[pair] || a.SKU != l.SKU || a.BatchCode == "" || a.Allocated <= 0 || !a.Eligible || a.EligibilityVersion <= 0 || !now.Before(a.Until) {
			return Fail("INVALID_ALLOCATION", 412)
		}
		slots[a.ID] = true
		batches[pair] = true
		sums[a.LineID] += int64(a.Allocated)
	}
	for _, l := range i.Lines {
		if sums[l.ID] != int64(l.Quantity) {
			return Fail("ALLOCATION_COVERAGE", 412)
		}
	}
	return nil
}

type Parcel struct {
	Weight  int32 `json:"weight_grams"`
	Length  int32 `json:"length_cm"`
	Width   int32 `json:"width_cm"`
	Height  int32 `json:"height_cm"`
	Fragile bool  `json:"fragile"`
}
type Package struct {
	ID         string  `json:"package_id"`
	State      string  `json:"state"`
	Seal       *string `json:"seal_code"`
	Parcel     *Parcel `json:"parcel"`
	EvidenceID *string `json:"primary_evidence_id"`
}
type Block struct {
	ID     string `json:"block_id"`
	Source string `json:"source"`
	Reason string `json:"reason_code"`
	Status string `json:"status"`
}
type Integration struct {
	Order    string `json:"order_source"`
	Shipping string `json:"shipping"`
}
type Task struct {
	ID                 string       `json:"task_id"`
	OrderID            string       `json:"order_id"`
	State              string       `json:"workflow_state"`
	Version            int64        `json:"version"`
	Generation         int64        `json:"ready_generation"`
	Warehouse          string       `json:"warehouse_id"`
	Assignee           *string      `json:"assignee_id"`
	Due                *time.Time   `json:"packing_due_at"`
	Admitted           time.Time    `json:"admitted_at"`
	Integration        Integration  `json:"integration"`
	Digest             string       `json:"items_digest"`
	PolicyVersion      string       `json:"policy_version"`
	Evidence           string       `json:"evidence_mode"`
	Lines              []Line       `json:"items"`
	Allocations        []Allocation `json:"allocations"`
	Checklist          []Checklist  `json:"checklist"`
	Blocks             []Block      `json:"active_blocks"`
	Package            Package      `json:"package"`
	AsOf               time.Time    `json:"queue_as_of"`
	Channel            string       `json:"channel"`
	LineCount          int          `json:"line_count"`
	Total              int64        `json:"total_quantity"`
	BlockCount         int          `json:"active_block_count"`
	Priority           string       `json:"priority_reason"`
	DeadlineRevision   int64        `json:"deadline_revision"`
	Inputs             Inputs       `json:"-"`
	Stock              Stock        `json:"-"`
	Cohort, Resume     string       `json:"-"`
	Started, Completed *time.Time   `json:"-"`
}

func (t *Task) Project(now time.Time) {
	t.AsOf = now
	t.LineCount = len(t.Lines)
	t.Total = 0
	for _, l := range t.Lines {
		t.Total += int64(l.Quantity)
	}
	t.BlockCount = len(t.Blocks)
	t.Priority = "NORMAL"
	if t.Due == nil {
		t.Priority = "SLA_UNAVAILABLE"
	} else if !now.Before(*t.Due) {
		t.Priority = "OVERDUE"
	} else if t.Due.Sub(now) < 30*time.Minute {
		t.Priority = "DUE_SOON"
	}
	if t.State == "BLOCKED" {
		t.Priority = "BLOCKED"
	}
	t.Integration = Integration{"PENDING", "PENDING"}
}

type Mutation struct {
	Action            string
	Expected          int64
	Child             string
	AssigneeID        string  `json:"assignee_id,omitempty"`
	Reason            string  `json:"reason,omitempty"`
	ReasonCode        string  `json:"reason_code,omitempty"`
	Counted           int32   `json:"counted_quantity,omitempty"`
	AllocationVersion int64   `json:"allocation_version,omitempty"`
	Result            string  `json:"result,omitempty"`
	Note              *string `json:"note,omitempty"`
	PackageID         string  `json:"package_id,omitempty"`
	Seal              string  `json:"seal_code,omitempty"`
	Parcel            *Parcel `json:"parcel,omitempty"`
	EvidenceID        *string `json:"primary_evidence_id,omitempty"`
}

func (t *Task) Permission(action string) string {
	switch action {
	case "AssignTask", "UnassignTask":
		return "fulfillment.task.assign_self"
	case "SealTask":
		return "fulfillment.task.seal"
	case "BlockTask":
		return "fulfillment.task.block"
	case "ResolveTaskBlock":
		return "fulfillment.task.resolve_block"
	case "ReopenTask":
		return "fulfillment.task.reopen"
	default:
		return "fulfillment.task.work"
	}
}
func (t *Task) Apply(p Principal, m Mutation, now time.Time) error {
	perm := t.Permission(m.Action)
	if (m.Action == "AssignTask" && m.AssigneeID != "" && m.AssigneeID != p.ID) || (m.Action == "UnassignTask" && (t.Assignee == nil || *t.Assignee != p.ID)) {
		perm = "fulfillment.task.assign_other"
	}
	if !p.Can(perm, t.Warehouse) {
		return Fail("FORBIDDEN", 403)
	}
	if m.Action == "ReopenTask" {
		return Fail("FEATURE_DISABLED", 409)
	}
	if t.Version != m.Expected {
		return Fail("VERSION_MISMATCH", 412)
	}
	assigned := t.Assignee != nil && *t.Assignee == p.ID
	switch m.Action {
	case "AssignTask":
		if t.State != "READY" {
			return Fail("TASK_ALREADY_ASSIGNED", 409)
		}
		id := p.ID
		if m.AssigneeID != "" {
			id = m.AssigneeID
		}
		if !UUID(id) {
			return Fail("INVALID_ASSIGNEE", 400)
		}
		if id != p.ID {
			return Fail("MEMBERSHIP_PROVIDER_UNAVAILABLE", 503)
		}
		t.Assignee = &id
		t.State = "ASSIGNED"
	case "UnassignTask":
		if t.State != "ASSIGNED" || len(strings.TrimSpace(m.Reason)) < 3 {
			return Fail("INVALID_UNASSIGN", 409)
		}
		t.Assignee = nil
		t.State = "READY"
	case "StartTask":
		if !assigned || t.State != "ASSIGNED" {
			return Fail("TASK_NOT_ASSIGNED", 409)
		}
		t.State = "IN_PROGRESS"
		t.Started = &now
	case "SetPickedQuantity":
		if !assigned || t.State != "IN_PROGRESS" {
			return Fail("TASK_NOT_ASSIGNED", 409)
		}
		found := false
		for n, a := range t.Allocations {
			if a.ID == m.Child {
				found = true
				if m.AllocationVersion != a.Version || m.Counted < 0 || m.Counted > a.Allocated || !a.Eligible || !now.Before(a.Until) {
					return Fail("INVALID_PICK", 409)
				}
				t.Allocations[n].Counted = m.Counted
			}
		}
		if !found {
			return Fail("ALLOCATION_NOT_FOUND", 404)
		}
	case "ConfirmChecklist":
		if !assigned || t.State != "IN_PROGRESS" || !contains([]string{"PASS", "FAIL"}, m.Result) {
			return Fail("INVALID_CHECKLIST", 409)
		}
		found := false
		for n, c := range t.Checklist {
			if c.Code == m.Child {
				found = true
				t.Checklist[n].Result = m.Result
				t.Checklist[n].By = &p.ID
				t.Checklist[n].At = &now
				t.Checklist[n].Note = m.Note
			}
		}
		if !found {
			return Fail("CHECKLIST_NOT_FOUND", 404)
		}
	case "BlockTask":
		if !contains([]string{"READY", "ASSIGNED", "IN_PROGRESS", "BLOCKED"}, t.State) || len(strings.TrimSpace(m.Reason)) < 3 || m.ReasonCode == "" {
			return Fail("INVALID_BLOCK", 409)
		}
		if t.State != "BLOCKED" {
			t.Resume = t.State
		}
		t.State = "BLOCKED"
		t.Blocks = append(t.Blocks, Block{uuid.NewString(), "OPERATOR", m.ReasonCode, "ACTIVE"})
	case "ResolveTaskBlock":
		if t.State != "BLOCKED" || len(strings.TrimSpace(m.Reason)) < 3 {
			return Fail("INVALID_RESOLUTION", 409)
		}
		found := false
		out := []Block{}
		for _, b := range t.Blocks {
			if b.ID == m.Child {
				found = true
				if b.Source != "OPERATOR" {
					return Fail("SOURCE_AUTHORITY_REQUIRED", 409)
				}
			} else {
				out = append(out, b)
			}
		}
		if !found {
			return Fail("BLOCK_NOT_FOUND", 404)
		}
		t.Blocks = out
		if len(out) == 0 {
			t.State = t.Resume
			t.Resume = ""
		}
	case "SealTask":
		if !assigned || t.State != "IN_PROGRESS" || len(t.Blocks) > 0 {
			return Fail("TASK_NOT_READY", 409)
		}
		if m.PackageID != t.Package.ID || m.Parcel == nil || !sealPattern.MatchString(m.Seal) {
			return Fail("INVALID_PACKAGE", 400)
		}
		x := m.Parcel
		pol := t.Inputs.Policy
		if x.Weight <= 0 || x.Weight > pol.MaxWeight || x.Length <= 0 || x.Width <= 0 || x.Height <= 0 || x.Length > pol.MaxDimension || x.Width > pol.MaxDimension || x.Height > pol.MaxDimension {
			return Fail("PARCEL_POLICY_CONFLICT", 409)
		}
		for _, l := range t.Lines {
			if l.Fragile && !x.Fragile {
				return Fail("FRAGILE_REQUIRED", 409)
			}
		}
		for _, c := range t.Checklist {
			if c.Required && c.Result != "PASS" {
				return Fail("CHECKLIST_INCOMPLETE", 409)
			}
		}
		for _, a := range t.Allocations {
			if a.Counted != a.Allocated || !a.Eligible || !now.Before(a.Until) {
				return Fail("PICK_MISMATCH", 409)
			}
		}
		t.State = "COMPLETED"
		t.Completed = &now
		t.Package.State = "SEALED"
		t.Package.Seal = &m.Seal
		t.Package.Parcel = x
		t.Package.EvidenceID = m.EvidenceID
	default:
		return fmt.Errorf("unknown action %s", m.Action)
	}
	t.Version++
	t.Project(now)
	return nil
}
