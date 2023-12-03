package actor

import "fmt"

type ErrInitFailed struct {
	Errors []error
}

func (e ErrInitFailed) Error() string {
	return fmt.Sprintf("failed to initialize engine: %v", e.Errors)
}
