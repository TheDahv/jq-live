package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"gitlab.com/thedahv/jq-live/json"
	"gitlab.com/thedahv/jq-live/ui"
)

func main() {
	var (
		compact   = flag.Bool("c", false, "compact output")
		debug     = flag.String("debug", "", "path to write debug information")
		raw       = flag.Bool("r", false, "raw output")
		debugFile *os.File
		program   string
		source    io.Reader
		jsonData  []byte
	)

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if *debug != "" {
		var err error
		debugFile, err = os.OpenFile(
			*debug,
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			0666,
		)
		if err != nil {
			log.Fatalf("unable to open debug file: %v", err)
		}
	}

	switch len(args) {
	case 2:
		// program + file path
		var err error
		program = args[0]
		if source, err = os.Open(args[1]); err != nil {
			log.Fatalf("unable to open input: %v", err)
		}
	case 1:
		// just the program
		program = args[0]
		source = os.Stdin
	case 0:
		// no program; assume object print as a default program and read from stdin
		program = "."
		source = os.Stdin
	default:
		flag.Usage()
		os.Exit(1)
	}

	processor, err := json.NewShell(
		json.OptionCompact(*compact),
		json.OptionRaw(*raw),
	)
	processor.Debug = debugFile
	if err != nil {
		log.Fatalf("unable to start up processor: %v", err)
	}

	if jsonData, err = ioutil.ReadAll(source); err != nil {
		log.Fatalf("unable to read json source data: %v", err)
	}

	// First parse, letting us know whether this is valid JSON or not
	out, err := processor.Process(bytes.NewReader(jsonData), program)
	if err != nil {
		log.Fatalf("unable to process JSON: %v", err)
	}

	display := &ui.Termbox{Debug: debugFile}
	if err := display.Start(program); err != nil {
		log.Fatalf("cannot start up display: %v", err)
	}

	// Initial draw
	display.RenderInput()
	if err := display.RenderResults(out); err != nil {
		log.Fatalf("cannot render result: %v", err)
	}

	// The UI display will emit action events on the channel representing actions
	// the application can take. Each can possibly be associated with an action to
	// update the internal state, followed by a render step.
	for {
		switch action := <-display.Events(); action {
		case ui.ActionBackspace:
			display.UpdateInputBackspace()

		case ui.ActionExit:
			display.Quit()
			os.Exit(0)

		case ui.ActionInput:
			display.UpdateInput()

		case ui.ActionPrint:
			display.Quit()
			out, err := processor.Process(bytes.NewReader(jsonData), display.Input)
			if err != nil {
				// TODO distinguish between normal parse errors and crashable errors
				if debugFile != nil {
					fmt.Fprintf(debugFile, "parse error: %v\n", err)
					fmt.Fprintf(debugFile, "program: %s\n", display.Input)
				}
				os.Exit(1)
			} else {
				io.Copy(os.Stdout, out)
				os.Exit(0)
			}

		case ui.ActionSave:
			// TODO
			// - Prompt for file input
			// - Accept prompt, or cancel save on empty input
			// - Write output to file and quit

		case ui.ActionToggleCompact:
			processor.ToggleCompact()
			out, err := processor.Process(bytes.NewReader(jsonData), display.Input)
			if err != nil {
				// TODO distinguish between normal parse errors and crashable errors
				if debugFile != nil {
					fmt.Fprintf(debugFile, "parse error: %v\n", err)
					fmt.Fprintf(debugFile, "program: %s\n", display.Input)
				}
			} else {
				err := display.RenderResults(out)
				if err != nil {
					log.Fatalf("cannot render result: %v", err)
				}
			}

		case ui.ActionToggleRaw:
			// TODO need some kind of modal indicator in the UI
			processor.ToggleRaw()
			out, err := processor.Process(bytes.NewReader(jsonData), display.Input)
			if err != nil {
				// TODO distinguish between normal parse errors and crashable errors
				if debugFile != nil {
					fmt.Fprintf(debugFile, "parse error: %v\n", err)
					fmt.Fprintf(debugFile, "program: %s\n", display.Input)
				}
			} else {
				err := display.RenderResults(out)
				if err != nil {
					log.Fatalf("cannot render result: %v", err)
				}
			}

		case ui.ActionSubmit:
			fmt.Fprintf(debugFile, "submitting program: %s\n", display.Input)
			out, err := processor.Process(bytes.NewReader(jsonData), display.Input)
			if err != nil {
				// TODO distinguish between normal parse errors and crashable errors
				if debugFile != nil {
					fmt.Fprintf(debugFile, "parse error: %v\n", err)
					fmt.Fprintf(debugFile, "program: %s\n", display.Input)
				}
			} else {
				err := display.RenderResults(out)
				if err != nil {
					log.Fatalf("cannot render result: %v", err)
				}
			}
		}

		display.RenderInput()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `jq-live: on-the-fly JSON processing

Usage: jq-live <program> [path]
`)
	flag.PrintDefaults()
}
