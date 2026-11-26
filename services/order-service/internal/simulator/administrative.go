package simulator

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/omamx/order-service/internal/domain"
)

// The administrative dataset is the canonical copy of the administrative
// units owned by profile-service (which seeds them from its migrations).
// The simulator validates checkout addresses against this dataset instead of
// a single hardcoded province/ward pair.
// TestAdministrativeDatasetMatchesProfileSeed fails the build if this copy
// ever diverges from the profile seed.
//
//go:embed administrative_units.json
var administrativeUnitsJSON []byte

type administrativeUnit struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Level      string  `json:"level"` // PROVINCE or WARD
	ParentCode *string `json:"parent_code"`
}

type administrativeDataset struct {
	Version string               `json:"version"`
	Units   []administrativeUnit `json:"units"`
}

// AdministrativeValidator checks province/ward pairs against the canonical
// dataset: the province must exist, the ward must exist, and the ward must
// belong to the province. It is the single validation policy shared by the
// address-validate, terms (quote) and revalidate (accept) provider paths.
type AdministrativeValidator struct {
	version   string
	provinces map[string]administrativeUnit
	wards     map[string]administrativeUnit
	checksum  string
}

func loadAdministrativeValidator(raw []byte) (AdministrativeValidator, error) {
	var ds administrativeDataset
	if e := json.Unmarshal(raw, &ds); e != nil {
		return AdministrativeValidator{}, fmt.Errorf("administrative dataset: %w", e)
	}
	v := AdministrativeValidator{
		version:   ds.Version,
		provinces: make(map[string]administrativeUnit),
		wards:     make(map[string]administrativeUnit),
	}
	for _, u := range ds.Units {
		switch u.Level {
		case "PROVINCE":
			if u.ParentCode != nil {
				return AdministrativeValidator{}, fmt.Errorf("administrative dataset: province %s has a parent", u.Code)
			}
			if _, dup := v.provinces[u.Code]; dup {
				return AdministrativeValidator{}, fmt.Errorf("administrative dataset: duplicate province %s", u.Code)
			}
			v.provinces[u.Code] = u
		case "WARD":
			if u.ParentCode == nil {
				return AdministrativeValidator{}, fmt.Errorf("administrative dataset: ward %s has no parent province", u.Code)
			}
			if _, dup := v.wards[u.Code]; dup {
				return AdministrativeValidator{}, fmt.Errorf("administrative dataset: duplicate ward %s", u.Code)
			}
			v.wards[u.Code] = u
		default:
			return AdministrativeValidator{}, fmt.Errorf("administrative dataset: unit %s has unknown level %q", u.Code, u.Level)
		}
	}
	if len(v.provinces) == 0 || len(v.wards) == 0 {
		return AdministrativeValidator{}, fmt.Errorf("administrative dataset: empty dataset")
	}
	for _, w := range v.wards {
		if _, ok := v.provinces[*w.ParentCode]; !ok {
			return AdministrativeValidator{}, fmt.Errorf("administrative dataset: ward %s references unknown province %s", w.Code, *w.ParentCode)
		}
	}
	lines := make([]string, 0, len(ds.Units))
	for _, u := range ds.Units {
		parent := ""
		if u.ParentCode != nil {
			parent = *u.ParentCode
		}
		lines = append(lines, u.Level+"|"+u.Code+"|"+parent)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	v.checksum = hex.EncodeToString(sum[:])
	return v, nil
}

var defaultAdministrativeValidator = func() AdministrativeValidator {
	v, e := loadAdministrativeValidator(administrativeUnitsJSON)
	if e != nil {
		panic(e)
	}
	return v
}()

func (s *Simulator) adminValidator() AdministrativeValidator {
	return defaultAdministrativeValidator
}

// Validate returns nil when the pair is a real ward of a real province in
// the canonical dataset, and a 422 INVALID_ADMINISTRATIVE_UNIT domain error
// otherwise.
func (v AdministrativeValidator) Validate(provinceCode, wardCode string) error {
	if _, ok := v.provinces[provinceCode]; !ok {
		return domain.E("INVALID_ADMINISTRATIVE_UNIT", 422)
	}
	w, ok := v.wards[wardCode]
	if !ok || w.ParentCode == nil || *w.ParentCode != provinceCode {
		return domain.E("INVALID_ADMINISTRATIVE_UNIT", 422)
	}
	return nil
}

func (v AdministrativeValidator) ProvinceName(code string) (string, bool) {
	u, ok := v.provinces[code]
	return u.Name, ok
}

func (v AdministrativeValidator) WardName(code string) (string, bool) {
	u, ok := v.wards[code]
	return u.Name, ok
}

func (v AdministrativeValidator) Version() string { return v.version }

// Checksum is the canonical SHA-256 of the dataset the validator actually
// loaded (level|code|parent records, sorted), so a deployment can be
// compared against the catalogue profile-service is serving.
func (v AdministrativeValidator) Checksum() string { return v.checksum }
