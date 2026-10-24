// Package jsonutil rejects ambiguous JSON before decoding typed API requests.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

func Decode(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("invalid JSON encoding")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("JSON object required")
	}
	tokens := json.NewDecoder(bytes.NewReader(data))
	tokens.UseNumber()
	if err := walk(tokens, 0); err != nil {
		return err
	}
	if _, err := tokens.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return checkNulls(data, reflect.TypeOf(target), 0)
}

// encoding/json silently turns null primitives into zero values. Pointer fields
// represent nullable schema fields; all other typed fields must reject null.
func checkNulls(raw json.RawMessage, t reflect.Type, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting limit exceeded")
	}
	if t == nil || t.Kind() == reflect.Interface {
		return nil
	}
	if t.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil
		}
		return checkNulls(raw, t.Elem(), depth+1)
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("null supplied for a non-nullable field")
	}
	if reflect.PointerTo(t).Implements(reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()) {
		return nil
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return err
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if tag == "-" {
				continue
			}
			if field.Anonymous && tag == "" {
				if err := checkNulls(raw, field.Type, depth+1); err != nil {
					return err
				}
				continue
			}
			if tag == "" {
				tag = field.Name
			}
			for name, value := range object {
				if strings.EqualFold(name, tag) {
					if err := checkNulls(value, field.Type, depth+1); err != nil {
						return err
					}
				}
			}
		}
	case reflect.Slice, reflect.Array:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		for _, item := range items {
			if err := checkNulls(item, t.Elem(), depth+1); err != nil {
				return err
			}
		}
	case reflect.Map:
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return err
		}
		for _, value := range object {
			if err := checkNulls(value, t.Elem(), depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func walk(d *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting limit exceeded")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return errors.New("invalid object key")
			}
			// encoding/json matches struct fields case-insensitively. Reject aliases too.
			folded := strings.ToLower(name)
			if keys[folded] {
				return errors.New("duplicate JSON field")
			}
			keys[folded] = true
			if err = walk(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := walk(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}
