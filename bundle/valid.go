package bundle

// valid is an interface with the checkValid function. This function should
// return an errors for incorrect data. It should be implemented for the
// different types and sub-types of a Bundle. Each type is able to check its
// sub-types and by tree-like calls all errors of a whole Bundle can be
// detected.
// For non-trivial code, the multierror package might be used.
type valid interface {
	// checkValid returns an array of errors for incorrect data.
	checkValid() error
}
