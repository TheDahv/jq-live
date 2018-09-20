package json

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
)

// Shell makes calls to the jq binary installed in the current environment to
// implement the Processor interface.
type Shell struct {
	Debug   io.Writer
	compact bool
}

// NewShell returns a new shell Processor with all configurations applied
func NewShell(opts ...ShellOption) (*Shell, error) {
	sh := &Shell{}

	var err error
	for _, opt := range opts {
		if sh, err = opt(sh); err != nil {
			return sh, err
		}
	}

	return sh, nil
}

// Process runs the input JSON and the processing program through the jq command
// with both as inputs via stdin. The results or a possible error are returned.
func (sh *Shell) Process(source io.Reader, program string) (io.Reader, error) {
	var args []string
	if sh.compact {
		args = append(args, "-c")
	}

	sh.debugf("processing program: %s\n", program)
	args = append(args, program)
	cmd := exec.Command("jq", args...)

	src, _ := ioutil.ReadAll(source)
	sh.debugf("file input:\n")
	sh.debugf(string(src))

	cmd.Stdin = bytes.NewReader(src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		sh.debugf("processing error: %v\n", err)
		return nil, fmt.Errorf("cannot read jq output: %v", err)
	}
	sh.debugf("program result:\n")
	sh.debugf(string(out))

	return bytes.NewReader(out), nil
}

// ShellOption allows a client to configure the behavior of the underlying jq
// process
type ShellOption func(*Shell) (*Shell, error)

// OptionCompact tells jq to return compact output
func OptionCompact(compact bool) ShellOption {
	return func(sh *Shell) (*Shell, error) {
		sh.compact = compact
		return sh, nil
	}
}

func (sh *Shell) debugf(format string, args ...interface{}) {
	if sh.Debug != nil {
		fmt.Fprintf(sh.Debug, "[Shell] "+format, args...)
	}
}
