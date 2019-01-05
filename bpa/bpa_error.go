package bpa

// BPAError is a simple error-struct with a msg string-field.
type BPAError struct {
	msg string
}

// newBPAError creates a new BPAError with the given message.
func newBPAError(msg string) *BPAError {
	return &BPAError{msg}
}

func (e BPAError) Error() string {
	return e.msg
}
