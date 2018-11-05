package bpa

// BPAError is a simple error-struct with a msg string-field.
type BPAError struct {
	msg string
}

// NewBPAError creates a new BPAError with the given message.
func NewBPAError(msg string) error {
	return &BPAError{msg}
}

func (e BPAError) Error() string {
	return e.msg
}
