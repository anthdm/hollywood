package actor

import "github.com/anthdm/hollywood/log"

// GetLogger returns the logger used by the engine, this is needed for the remote
// to be able to log. It shouldn't be used by anything else.
func (e *Engine) GetLogger() log.Logger {
	return e.logger
}
