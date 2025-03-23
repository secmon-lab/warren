package ptr

// Ref returns a pointer to the given value.
func Ref[T any](v T) *T {
	return &v
}

// Deref returns the value pointed to by the given pointer.
func Deref[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}
