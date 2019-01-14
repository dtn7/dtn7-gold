package bpa

// bpaError is a simple error-struct.
type bpaError struct {
	msg string
}

// newBpaError creates a new bpaError with the given message.
func newBpaError(msg string) *bpaError {
	return &bpaError{msg}
}

func (e bpaError) Error() string {
	return e.msg
}
