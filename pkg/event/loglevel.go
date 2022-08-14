package event

// LogLevel indicates the severity of the event message:
//   - TRACE: The finest-grained, step-by-step informational events
//   - DEBUG: Fine-grained informational events that are most useful to debug an application
//   - INFO: Informational messages that highlight the progress of the application at a coarse-grained level
//   - WARN: Potentially harmful situations
//   - ERROR: Error events that might still allow the application to continue running
//   - FATAL: Very severe error events that will presumably lead the application to abort
type LogLevel string

// TRACE events are the finest-grained, step-by-step informational events
const TRACE LogLevel = "TRACE"

// DEBUG events are fine-grained informational events that are most useful to debug an application
const DEBUG LogLevel = "DEBUG"

// INFO events are informational messages that highlight the progress of the application at a coarse-grained level
const INFO LogLevel = "INFO"

// WARN events indicate potentially harmful situations
const WARN LogLevel = "WARN"

// ERROR events indicate error events that might still allow the application to continue running
const ERROR LogLevel = "ERROR"

// FATAL events indicate very severe error events that will presumably lead the application to abort
const FATAL LogLevel = "FATAL"

// LogLevels is the complete list of valid LogLevels
var LogLevels = []LogLevel{TRACE, DEBUG, INFO, WARN, ERROR, FATAL}

// IsValid returns true if the supplied LogLevel is recognized
func (l LogLevel) IsValid() bool {
	for _, v := range LogLevels {
		if l == v {
			return true
		}
	}
	return false
}
