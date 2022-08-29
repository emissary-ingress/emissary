package agent

// StrToPointer will return the pointer to the given string.
func StrToPointer(str string) *string {
	return &str
}

// Float64ToPointer will return the pointer to the given float.
func Float64ToPointer(f float64) *float64 {
	return &f
}

// Int64ToPointer will return the pointer to the given int64.
func Int64ToPointer(i int64) *int64 {
	return &i
}
