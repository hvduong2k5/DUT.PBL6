package migrations

import (
	"strings"
	"testing"
)

func TestMigrationVersionsAreUniqueAndHaveDownFiles(t *testing.T) {
	entries, err := files.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	versions := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 || len(parts[0]) != 6 {
			t.Fatalf("migration %q must start with a six-digit version and underscore", name)
		}
		for _, digit := range parts[0] {
			if digit < '0' || digit > '9' {
				t.Fatalf("migration %q version must be numeric", name)
			}
		}
		if previous, exists := versions[parts[0]]; exists {
			t.Fatalf("migration version %s is reused by %s and %s", parts[0], previous, name)
		}
		versions[parts[0]] = name
		if _, err := files.ReadFile(strings.TrimSuffix(name, ".up.sql") + ".down.sql"); err != nil {
			t.Fatalf("migration %s is missing its down file: %v", name, err)
		}
	}
}
