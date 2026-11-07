package httpapi

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeJSONRejectsBodyLargerThanOneMiB(t *testing.T) {
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"value":"`+strings.Repeat("a", 1<<20)+`"}`))
	response := httptest.NewRecorder()

	var payload struct {
		Value string `json:"value"`
	}
	err := decodeJSON(response, request, &payload)

	require.Error(t, err)
	require.Contains(t, err.Error(), "request body too large")
}

func TestDecodeJSONRejectsTrailingJSONValue(t *testing.T) {
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"value":"first"}{"value":"second"}`))
	response := httptest.NewRecorder()

	var payload struct {
		Value string `json:"value"`
	}
	err := decodeJSON(response, request, &payload)

	require.Error(t, err)
}
