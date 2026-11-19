package domain

import (
	"math"
	"testing"
)

func TestDigestCanonicalAndOverflow(t *testing.T) {
	got, e := Digest([]Line{{SKU: "MX-GION-500G", Quantity: 1}, {SKU: "MX-GION-500G", Quantity: 1}})
	if e != nil || got != "a70dd9eecf2ba6537a9027f1d7974ddea8ffa8f0d73b88d3f0ff69cc46472208" {
		t.Fatalf("%s %v", got, e)
	}
	for _, lines := range [][]Line{{}, {{SKU: "A", Quantity: 0}}, {{SKU: "A", Quantity: math.MaxInt32}, {SKU: "A", Quantity: 1}}, {{SKU: "sản-phẩm", Quantity: 1}}} {
		if _, e = Digest(lines); e == nil {
			t.Fatal("invalid quantities/SKU accepted")
		}
	}
}
func TestKeysAndCanonicalNestedObjects(t *testing.T) {
	for _, k := range []string{"short", "with space in key!", string(make([]byte, 20))} {
		if Key(k) {
			t.Fatal("invalid key accepted")
		}
	}
	if !Key("command-key-0000001") {
		t.Fatal("valid key rejected")
	}
	a := struct {
		Z int `json:"z"`
		A int `json:"a"`
	}{1, 2}
	if Hash(map[string]any{"body": a}) != Hash(map[string]any{"body": map[string]any{"a": 2, "z": 1}}) {
		t.Fatal("nested object normalization differs")
	}
}
