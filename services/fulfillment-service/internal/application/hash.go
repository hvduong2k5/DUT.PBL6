package application

import (
	"fmt"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func ProtoHash(m proto.Message) (string, error) {
	if m == nil {
		return "", d.Fail("EMPTY_MESSAGE", 400)
	}
	v, e := canonical(m.ProtoReflect())
	if e != nil {
		return "", e
	}
	return d.Hash(v), nil
}
func canonical(m protoreflect.Message) (map[string]any, error) {
	out := map[string]any{}
	fields := m.Descriptor().Fields()
	for n := 0; n < fields.Len(); n++ {
		f := fields.Get(n)
		name := string(f.Name())
		if name == "request_hash" || name == "idempotency_key" || name == "correlation_id" || !m.Has(f) {
			continue
		}
		if f.IsMap() {
			return nil, fmt.Errorf("unsupported canonical map")
		}
		v := m.Get(f)
		if f.IsList() {
			a := []any{}
			for k := 0; k < v.List().Len(); k++ {
				x, e := scalar(f, v.List().Get(k))
				if e != nil {
					return nil, e
				}
				a = append(a, x)
			}
			if len(a) > 0 {
				out[name] = a
			}
			continue
		}
		x, e := scalar(f, v)
		if e != nil {
			return nil, e
		}
		if y, ok := x.(map[string]any); ok && len(y) == 0 {
			continue
		}
		out[name] = x
	}
	return out, nil
}
func scalar(f protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	switch f.Kind() {
	case protoreflect.MessageKind:
		return canonical(v.Message())
	case protoreflect.StringKind:
		return v.String(), nil
	case protoreflect.BoolKind:
		return v.Bool(), nil
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return v.Int(), nil
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return v.Uint(), nil
	case protoreflect.EnumKind:
		return int32(v.Enum()), nil
	}
	return nil, fmt.Errorf("unsupported hash kind %s", f.Kind())
}
