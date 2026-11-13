package grpcapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

var excludedHashFields = map[protoreflect.Name]struct{}{
	"request_hash":    {},
	"correlation_id":  {},
	"idempotency_key": {},
}

func canonicalMessageHash(message protoreflect.Message) (string, error) {
	value, err := canonicalMessage(message)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("encode canonical request: %w", err)
	}
	encoded := strings.TrimSuffix(buffer.String(), "\n")
	digest := sha256.Sum256([]byte(encoded))
	return hex.EncodeToString(digest[:]), nil
}

func canonicalMessage(message protoreflect.Message) (map[string]any, error) {
	result := make(map[string]any)
	fields := message.Descriptor().Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		if _, excluded := excludedHashFields[field.Name()]; excluded {
			continue
		}
		if !message.Has(field) {
			continue
		}
		value, include, err := canonicalField(field, message.Get(field))
		if err != nil {
			return nil, err
		}
		if include {
			result[string(field.Name())] = value
		}
	}
	return result, nil
}

func canonicalField(field protoreflect.FieldDescriptor, value protoreflect.Value) (any, bool, error) {
	if field.IsList() {
		list := value.List()
		if list.Len() == 0 {
			return nil, false, nil
		}
		items := make([]any, 0, list.Len())
		for index := 0; index < list.Len(); index++ {
			item, _, err := canonicalScalar(field, list.Get(index))
			if err != nil {
				return nil, false, err
			}
			items = append(items, item)
		}
		return items, true, nil
	}
	if field.IsMap() {
		return nil, false, fmt.Errorf("canonical protobuf maps are not supported")
	}
	return canonicalScalar(field, value)
}

func canonicalScalar(field protoreflect.FieldDescriptor, value protoreflect.Value) (any, bool, error) {
	switch field.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		nested, err := canonicalMessage(value.Message())
		return nested, len(nested) > 0, err
	case protoreflect.StringKind:
		return value.String(), field.HasPresence() || value.String() != "", nil
	case protoreflect.BoolKind:
		return value.Bool(), field.HasPresence() || value.Bool(), nil
	case protoreflect.EnumKind:
		number := int32(value.Enum())
		return number, field.HasPresence() || number != 0, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		number := int32(value.Int())
		return number, field.HasPresence() || number != 0, nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		number := value.Int()
		return number, field.HasPresence() || number != 0, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		number := uint32(value.Uint())
		return number, field.HasPresence() || number != 0, nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		number := value.Uint()
		return number, field.HasPresence() || number != 0, nil
	default:
		return nil, false, fmt.Errorf("field %s uses unsupported canonical kind %s", field.FullName(), field.Kind())
	}
}
