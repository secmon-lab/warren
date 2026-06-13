package test

import (
	"reflect"
	"testing"
	"unsafe"
)

// PrivateField reads an unexported struct field by name via reflection.
//
// It is intended for tests that need to verify the effect of functional options
// (e.g. gollem-dev/tools `WithXxx` helpers) on an external ToolSet whose fields
// are unexported. structPtr must be a non-nil pointer to a struct. The returned
// value is the field value as `any`; callers assert the concrete type with gt.
func PrivateField(t *testing.T, structPtr any, name string) any {
	t.Helper()

	rv := reflect.ValueOf(structPtr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		t.Fatalf("PrivateField: structPtr must be a non-nil pointer to a struct, got %T", structPtr)
	}

	fv := rv.Elem().FieldByName(name)
	if !fv.IsValid() {
		t.Fatalf("PrivateField: no such field %q on %T", name, structPtr)
	}

	// The field is unexported, so make an addressable, readable copy via the
	// field's address. structPtr is a pointer, so the field is addressable.
	return reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem().Interface()
}
