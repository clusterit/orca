package errors

import "gopkg.in/errgo.v1"

var (
	notFound = errgo.New("Not found")
)

// NotFound returns a notfound error
func NotFound(e error, f string, a ...interface{}) error {
	return errgo.WithCausef(e, notFound, f, a...)
}

// IsNotFound checks if an error is a notfound error
func IsNotFound(e error) bool {
	return errgo.Cause(e) == notFound
}
