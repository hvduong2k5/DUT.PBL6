package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omamx/procurement-service/internal/domain"
)

func TestMapErrorClassifiesConstraintFailures(t *testing.T) {
	tests := []struct {
		name string
		code string
		want error
	}{
		{name: "duplicate key", code: "23505", want: domain.ErrConflict},
		{name: "foreign key", code: "23503", want: domain.ErrInvalidArgument},
		{name: "check constraint", code: "23514", want: domain.ErrInvalidArgument},
		{name: "invalid text representation", code: "22P02", want: domain.ErrInvalidArgument},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapError(&pgconn.PgError{Code: tc.code, ConstraintName: "test_constraint", Message: "rejected"})
			if !errors.Is(got, tc.want) {
				t.Fatalf("mapError() = %v, want errors.Is(_, %v)", got, tc.want)
			}
		})
	}
}

func TestMapErrorPreservesUnexpectedErrors(t *testing.T) {
	want := errors.New("database unavailable")
	if got := mapError(want); !errors.Is(got, want) {
		t.Fatalf("mapError() = %v, want original error", got)
	}
}
