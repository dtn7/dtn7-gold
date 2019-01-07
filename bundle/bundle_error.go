package bundle

// bundleError is a simple error-struct.
type bundleError struct {
	msg string
}

// newBundleError creates a new bundleError with the given message.
func newBundleError(msg string) *bundleError {
	return &bundleError{msg}
}

func (e bundleError) Error() string {
	return e.msg
}
