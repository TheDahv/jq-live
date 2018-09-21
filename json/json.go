// Package json manages applying a program of selectors, processors, and
// extractors as defined by the jq program against input JSON and producing the
// result.
package json

import "io"

// Processor runs a set of commands over some input JSON and outputs the result
type Processor interface {
	Process(source io.Reader, program string) (io.Reader, error)
}
