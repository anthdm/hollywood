package actor

import "fmt"

type ErrInitFailed struct {
	Errors []error
}

func (e ErrInitFailed) Error() string {
	// Todo: make the error pretty
	return fmt.Sprintf("failed to initialize engine: %v", e.Errors)
}
