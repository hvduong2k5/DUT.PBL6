package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// parseJSON rejects ambiguous input before Go's ordinary decoder can discard
// duplicate keys, replace invalid Unicode, or round integers through float64.
func parseJSON(data []byte) (any, error) {
	if len(data) == 0 || len(data) > 1<<20 || !utf8.Valid(data) || !validSurrogates(data) {
		return nil, ErrSnapshot
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	v, err := readValue(d, 0)
	if err != nil {
		return nil, ErrSnapshot
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrSnapshot
	}
	return v, nil
}

func readValue(d *json.Decoder, depth int) (any, error) {
	if depth > 32 {
		return nil, ErrSnapshot
	}
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch v := token.(type) {
	case json.Delim:
		switch v {
		case '{':
			obj := make(map[string]any)
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				name, ok := key.(string)
				if !ok || !norm.NFC.IsNormalString(name) {
					return nil, ErrSnapshot
				}
				if _, exists := obj[name]; exists {
					return nil, ErrSnapshot
				}
				value, err := readValue(d, depth+1)
				if err != nil {
					return nil, err
				}
				obj[name] = value
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return nil, ErrSnapshot
			}
			return obj, nil
		case '[':
			arr := make([]any, 0)
			for d.More() {
				value, err := readValue(d, depth+1)
				if err != nil {
					return nil, err
				}
				arr = append(arr, value)
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return nil, ErrSnapshot
			}
			return arr, nil
		}
	case json.Number:
		// Exponents, fractions and negative zero are not the integer profile.
		i, err := strconv.ParseInt(v.String(), 10, 64)
		if err != nil || strconv.FormatInt(i, 10) != v.String() {
			return nil, ErrSnapshot
		}
		return v, nil
	case string:
		if !norm.NFC.IsNormalString(v) {
			return nil, ErrSnapshot
		}
		return v, nil
	case bool, nil:
		return v, nil
	}
	return nil, ErrSnapshot
}

// The standard JSON decoder replaces lone escaped UTF-16 surrogates with U+FFFD.
// Reject them instead of hashing silently repaired commercial input.
func validSurrogates(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '"' {
			continue
		}
		i++
		for i < len(data) && data[i] != '"' {
			if data[i] != '\\' {
				i++
				continue
			}
			i++
			if i >= len(data) {
				return false
			}
			if data[i] != 'u' {
				i++
				continue
			}
			if i+4 >= len(data) {
				return false
			}
			u, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
			if err != nil {
				return false
			}
			i += 5
			if u >= 0xdc00 && u <= 0xdfff {
				return false
			}
			if u >= 0xd800 && u <= 0xdbff {
				if i+5 >= len(data) || data[i] != '\\' || data[i+1] != 'u' {
					return false
				}
				low, err := strconv.ParseUint(string(data[i+2:i+6]), 16, 16)
				if err != nil || low < 0xdc00 || low > 0xdfff {
					return false
				}
				i += 6
			}
		}
	}
	return true
}

func canonicalJSON(value any) ([]byte, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, ErrSnapshot
	}
	v, err := parseJSON(b)
	if err != nil {
		return nil, err
	}
	return canonicalValue(v)
}

func canonicalValue(v any) ([]byte, error) {
	var b bytes.Buffer
	var write func(any) error
	write = func(value any) error {
		switch x := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(x))
			for key := range x {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			b.WriteByte('{')
			for i, key := range keys {
				if i > 0 {
					b.WriteByte(',')
				}
				writeString(&b, key)
				b.WriteByte(':')
				if err := write(x[key]); err != nil {
					return err
				}
			}
			b.WriteByte('}')
		case []any:
			b.WriteByte('[')
			for i, item := range x {
				if i > 0 {
					b.WriteByte(',')
				}
				if err := write(item); err != nil {
					return err
				}
			}
			b.WriteByte(']')
		case string:
			writeString(&b, x)
		case json.Number:
			b.WriteString(x.String())
		case nil:
			b.WriteString("null")
		case bool:
			b.WriteString(strconv.FormatBool(x))
		default:
			return ErrSnapshot
		}
		return nil
	}
	if err := write(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(c)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 {
				fmt.Fprintf(b, `\u%04x`, c)
			} else {
				b.WriteRune(c)
			}
		}
	}
	b.WriteByte('"')
}

// Every snapshot field is required, including explicit nulls for pointers. Go's
// zero-value decoding alone cannot distinguish an omitted discount from zero.
func requireShape(v any, t reflect.Type) error {
	if t.Kind() == reflect.Pointer {
		if v == nil {
			return nil
		}
		return requireShape(v, t.Elem())
	}
	if t == reflect.TypeOf(time.Time{}) {
		if _, ok := v.(string); !ok {
			return ErrSnapshot
		}
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		obj, ok := v.(map[string]any)
		if !ok || len(obj) != t.NumField() {
			return ErrSnapshot
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			item, exists := obj[field.Tag.Get("json")]
			if !exists {
				return ErrSnapshot
			}
			if err := requireShape(item, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		arr, ok := v.([]any)
		if !ok {
			return ErrSnapshot
		}
		for _, item := range arr {
			if err := requireShape(item, t.Elem()); err != nil {
				return err
			}
		}
	case reflect.String:
		if _, ok := v.(string); !ok {
			return ErrSnapshot
		}
	case reflect.Int, reflect.Int64:
		if _, ok := v.(json.Number); !ok {
			return ErrSnapshot
		}
	default:
		return ErrSnapshot
	}
	return nil
}

func validInputStrings(v reflect.Value) bool {
	if v.Type() == reflect.TypeOf(time.Time{}) {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer:
		return v.IsNil() || validInputStrings(v.Elem())
	case reflect.String:
		return utf8.ValidString(v.String()) && norm.NFC.IsNormalString(v.String())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !validInputStrings(v.Field(i)) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if !validInputStrings(v.Index(i)) {
				return false
			}
		}
	}
	return true
}
