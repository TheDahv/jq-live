package json

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// Shell makes calls to the jq binary installed in the current environment to
// implement the Processor interface.
type Shell struct {
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

	args = append(args, program)
	cmd := exec.Command("jq", args...)
	cmd.Stdin = source
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read jq output: %v", err)
	}

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
