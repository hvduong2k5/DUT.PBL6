package kafka

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMonitoringSharedAdminContract(t *testing.T) {
	raw, err := os.ReadFile("../../../../../docs/03_api_specs/admin-stable.openapi.yaml")
	require.NoError(t, err)
	var doc struct {
		Paths map[string]struct {
			Get struct {
				Security []map[string][]string `yaml:"security"`
				Servers  []struct {
					URL string `yaml:"url"`
				} `yaml:"servers"`
				Responses map[string]struct {
					Content map[string]struct {
						Schema map[string]any `yaml:"schema"`
					} `yaml:"content"`
				} `yaml:"responses"`
			} `yaml:"get"`
		} `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	defer procurementHealthy.Store(false)
	for _, tc := range []struct {
		path             string
		enabled, healthy bool
	}{
		{"/readyz", false, false}, {"/readyz", true, true}, {"/readyz", true, false}, {"/metrics", true, true},
	} {
		procurementHealthy.Store(tc.healthy)
		response := httptest.NewRecorder()
		ProcurementMonitor(tc.enabled).ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil))
		path, ok := doc.Paths[tc.path]
		require.True(t, ok)
		require.NotNil(t, path.Get.Security)
		require.Empty(t, path.Get.Security)
		require.Len(t, path.Get.Servers, 1)
		require.Equal(t, "http://127.0.0.1:8011", path.Get.Servers[0].URL)
		declared, ok := path.Get.Responses[strconv.Itoa(response.Code)]
		require.True(t, ok, "undocumented HTTP status")
		contentType := strings.Split(response.Result().Header.Get("Content-Type"), ";")[0]
		content, ok := declared.Content[contentType]
		require.True(t, ok, "undocumented content type")
		if response.Code == 503 {
			require.Equal(t, "#/components/schemas/Error", content.Schema["$ref"])
			var failure map[string]string
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &failure))
			for _, key := range []string{"code", "message", "requestId"} {
				require.NotEmpty(t, failure[key])
			}
		}
	}
}
