package jsonutil

import (
	"strings"
	"testing"
)

func TestStrictObjects(t *testing.T) {
	type request struct {
		Value    int            `json:"value"`
		Nested   map[string]any `json:"nested"`
		Enabled  bool           `json:"enabled"`
		Items    []string       `json:"items"`
		Optional *string        `json:"optional"`
	}
	for _, raw := range []string{`null`, `[]`, `{"value":null}`, `{"enabled":null}`, `{"items":null}`, `{"items":[null]}`, `{"value":1} {}`, `{"value":1,"value":2}`, `{"value":1,"VALUE":2}`, `{"unknown":true}`, `{"nested":{"a":1,"\u0061":2}}`, `{"value":1.5}`, `{"nested":` + strings.Repeat("[", 66) + `0` + strings.Repeat("]", 66) + `}`, "{\"value\":\"\xff\"}"} {
		var target request
		if err := Decode([]byte(raw), &target); err == nil {
			t.Errorf("ambiguous JSON accepted: %s", raw)
		}
	}
	var target request
	if err := Decode([]byte(`{"value":2,"optional":null,"nested":{"nullable":null,"list":[1,{"a":true}],"text":"valid"}}`), &target); err != nil || target.Value != 2 {
		t.Fatalf("valid JSON: %+v %v", target, err)
	}
}
