package core

// coreError is a simple error-struct.
type coreError struct {
	msg string
}

// newCoreError creates a new coreError with the given message.
func newCoreError(msg string) *coreError {
	return &coreError{msg}
}

func (e coreError) Error() string {
	return e.msg
}
