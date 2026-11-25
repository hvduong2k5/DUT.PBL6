package simulator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The canonical dataset embedded in this package must stay identical to the
// administrative_units catalogue that profile-service (the data owner)
// serves. Since migration 000005 the catalogue is defined by a single
// full-replace migration (DELETE + INSERT of the whole set), so the
// expected content is exactly that file's rows; the 000001/000002 seeds
// are superseded history. This is the candidate-consistency check from
// the address-seam design: divergence fails the build in the Dev Gate
// instead of surfacing as a production checkout failure.
func TestAdministrativeDatasetMatchesProfileSeed(t *testing.T) {
	root := repoRoot(t)
	files := []string{
		filepath.Join(root, "services", "profile-service", "migrations", "000005_full_administrative_catalogue.up.sql"),
	}
	row := regexp.MustCompile(`\('([^']+)',\s*'([^']*)',\s*'[^']*',\s*'[^']*',\s*(NULL|'[^']*'),\s*'(PROVINCE|WARD)'\)`)
	want := map[string]string{}
	for _, f := range files {
		b, e := os.ReadFile(f)
		if e != nil {
			t.Fatalf("read profile seed %s: %v", f, e)
		}
		for _, m := range row.FindAllStringSubmatch(string(b), -1) {
			parent := ""
			if m[3] != "NULL" {
				parent = strings.Trim(m[3], "'")
			}
			want[m[4]+"|"+m[1]+"|"+parent] = m[2]
		}
	}
	if len(want) == 0 {
		t.Fatal("no administrative units parsed from the profile seed")
	}
	v := defaultAdministrativeValidator
	got := map[string]string{}
	for _, u := range v.provinces {
		got["PROVINCE|"+u.Code+"|"] = u.Name
	}
	for _, u := range v.wards {
		got["WARD|"+u.Code+"|"+*u.ParentCode] = u.Name
	}
	if len(got) != len(want) {
		t.Fatalf("dataset size mismatch: embedded %d units, profile seed %d units", len(got), len(want))
	}
	for key, name := range want {
		if got[key] != name {
			t.Errorf("unit %s: embedded name %q, profile seed name %q", key, got[key], name)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, e := os.Getwd()
	if e != nil {
		t.Fatal(e)
	}
	for {
		probe := filepath.Join(wd, "services", "profile-service", "migrations", "000001_init_schema.up.sql")
		if _, e := os.Stat(probe); e == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatalf("repo root not found walking up from the package directory")
		}
		wd = parent
	}
}
