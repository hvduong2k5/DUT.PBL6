package observability

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPrometheusRulesHaveRequiredFields(t *testing.T) {
	contents, err := os.ReadFile("prometheus-rules.yaml")
	require.NoError(t, err)

	var rules struct {
		Groups []struct {
			Name  string `yaml:"name"`
			Rules []struct {
				Alert string `yaml:"alert"`
				Expr  string `yaml:"expr"`
			} `yaml:"rules"`
		} `yaml:"groups"`
	}
	require.NoError(t, yaml.Unmarshal(contents, &rules))
	require.NotEmpty(t, rules.Groups)
	for _, group := range rules.Groups {
		require.NotEmpty(t, group.Name)
		require.NotEmpty(t, group.Rules)
		for _, rule := range group.Rules {
			require.NotEmpty(t, rule.Alert)
			require.NotEmpty(t, rule.Expr)
		}
	}
}

func TestGrafanaDashboardIsValidJSON(t *testing.T) {
	contents, err := os.ReadFile("grafana-dashboard.json")
	require.NoError(t, err)
	var dashboard struct {
		Title  string `json:"title"`
		UID    string `json:"uid"`
		Panels []struct {
			ID      int    `json:"id"`
			Title   string `json:"title"`
			Targets []struct {
				Expression string `json:"expr"`
			} `json:"targets"`
		} `json:"panels"`
	}
	require.NoError(t, json.Unmarshal(contents, &dashboard))
	require.Equal(t, "omama-care-service", dashboard.UID)
	require.NotEmpty(t, dashboard.Title)
	require.NotEmpty(t, dashboard.Panels)
	for _, panel := range dashboard.Panels {
		require.Positive(t, panel.ID)
		require.NotEmpty(t, panel.Title)
		require.NotEmpty(t, panel.Targets)
		for _, target := range panel.Targets {
			require.NotEmpty(t, target.Expression)
		}
	}
}
