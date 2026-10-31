package testutil

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RuntimeDatabase returns a fresh LOGIN credential for the test's own schema.
// Never print the returned URL: it contains an ephemeral password.
func RuntimeDatabase(t *testing.T) (context.Context, string, *pgxpool.Pool) {
	t.Helper()
	ctx, admin := Database(t)
	var schema string
	if err := admin.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	role := schema + "_runtime"
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	password := base64.RawURLEncoding.EncodeToString(secret)
	var statement string
	if err := admin.QueryRow(ctx, `SELECT format('CREATE ROLE %I LOGIN NOINHERIT PASSWORD %L',$1::text,$2::text)`, role, password).Scan(&statement); err != nil {
		t.Fatal("cannot prepare runtime role")
	}
	if _, err := admin.Exec(ctx, statement); err != nil {
		t.Fatal("cannot create runtime test role")
	}
	t.Cleanup(func() {
		clean, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(clean, "DROP OWNED BY "+pgx.Identifier{role}.Sanitize()); err != nil {
			t.Error("runtime grant cleanup failed")
		}
		if _, err := admin.Exec(clean, "DROP ROLE "+pgx.Identifier{role}.Sanitize()); err != nil {
			t.Error("runtime role cleanup failed")
		}
	})
	conn, err := admin.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = migrations.GrantRuntime(ctx, conn.Conn(), role)
	conn.Release()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(os.Getenv("TRACEABILITY_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal("invalid test URL")
	}
	parsed.User = url.UserPassword(role, password)
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return ctx, parsed.String(), admin
}
