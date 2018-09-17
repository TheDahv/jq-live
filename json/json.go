package json

import "io"

// Processor runs a set of commands over some input JSON and outputs the result
type Processor interface {
	Process(source io.Reader, program string) (io.Reader, error)
}
