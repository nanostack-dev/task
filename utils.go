package task

// PointerOrDefaultValue returns the value of a pointer or a default value.
func PointerOrDefaultValue[T any](value *T, defaultValue T) T {
	if value == nil {
		return defaultValue
	}
	return *value
}

func PointerOrDefaultPointer[T any](value *T, defaultValue *T) *T {
	if value == nil {
		return defaultValue
	}
	return value
}

// ToPointer converts a value to a pointer.
func ToPointer[T any](value T) *T {
	return &value
}
