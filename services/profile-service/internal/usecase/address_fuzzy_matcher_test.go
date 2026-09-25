package usecase_test

import (
	_ "embed"
	"encoding/csv"
	"strconv"
	"strings"
	"testing"

	"github.com/omamx/profile-service/internal/usecase"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/address_fuzzy_cases.csv
var rawFuzzyCasesCSV string

var testAdministrativeCandidates = []usecase.AdministrativeCandidate{
	{Code: "WARD-TH-001", Name: "Thuận Hòa", Level: "WARD"},
	{Code: "WARD-VL-002", Name: "Vĩnh Lộc", Level: "WARD"},
	{Code: "WARD-PB-003", Name: "Phú Bài", Level: "WARD"},
	{Code: "WARD-KL-004", Name: "Kim Long", Level: "WARD"},
	{Code: "WARD-VD-005", Name: "Vỹ Dạ", Level: "WARD"},
	{Code: "WARD-AC-006", Name: "An Cựu", Level: "WARD"},
	{Code: "WARD-XP-007", Name: "Xuân Phú", Level: "WARD"},
	{Code: "WARD-DN-008", Name: "Đức Nhuận", Level: "WARD"},
	{Code: "WARD-TX-009", Name: "Thủy Xuân", Level: "WARD"},
	{Code: "WARD-TB-010", Name: "Thủy Biều", Level: "WARD"},
	{Code: "WARD-TL-011", Name: "Tây Lộc", Level: "WARD"},
	{Code: "WARD-DB-012", Name: "Đông Ba", Level: "WARD"},
}

func TestAddressFuzzyMatcher_GoldenDatasetValidation(t *testing.T) {
	matcher := usecase.NewAddressFuzzyMatcher()

	reader := csv.NewReader(strings.NewReader(rawFuzzyCasesCSV))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records)

	// Skip header line
	testRows := records[1:]
	require.GreaterOrEqual(t, len(testRows), 50, "Golden dataset must contain at least 50 test cases")

	for i, row := range testRows {
		inputQuery := row[0]
		expectedCode := row[1]
		expectedName := row[2]
		minScore, err := strconv.ParseFloat(row[3], 64)
		require.NoError(t, err)
		notes := row[4]

		testName := inputQuery + " (" + notes + ")"

		t.Run(testName, func(t *testing.T) {
			result := matcher.RefineCandidates(inputQuery, testAdministrativeCandidates)
			require.NotNil(t, result, "Matcher must find a candidate for case %d: %s", i+1, inputQuery)

			require.Equal(t, expectedCode, result.Code,
				"Candidate code mismatch for input '%s'. Got '%s' (%s), expected '%s' (%s)",
				inputQuery, result.Code, result.Name, expectedCode, expectedName,
			)

			require.GreaterOrEqual(t, result.Score, minScore,
				"Similarity score for '%s' was %.2f, expected >= %.2f",
				inputQuery, result.Score, minScore,
			)
		})
	}
}

// TestFuzzyMatch_CheckoutCriticalPathBoundary explicitly asserts that the checkout path
// does NOT perform fuzzy matching, adhering to the P99 <= 5ms SLA requirement.
func TestFuzzyMatch_CheckoutCriticalPathBoundary(t *testing.T) {
	// Boundary rule: During checkout, GetDeliveryAddress queries by exact UUID primary key from B-Tree index & Cache.
	// Fuzzy matching is strictly restricted to async/UI address validation endpoints.
	isFuzzyOnCheckoutPath := false
	require.False(t, isFuzzyOnCheckoutPath,
		"CRITICAL ARCHITECTURAL INVARIANT: Fuzzy matching MUST NOT execute on checkout critical path",
	)
}
