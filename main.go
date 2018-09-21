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
		source    io.Reader
		jsonData  []byte
	)

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("unable to determine current directory: %v", err)
	}

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

	var program = "."
	if inputOnStdin(os.Stdin) {
		source = os.Stdin
		if len(args) > 0 {
			program = args[0]
		}
	} else {
		switch len(args) {
		case 2:
			// program + file path
			var err error
			program = args[0]
			if source, err = os.Open(args[1]); err != nil {
				log.Fatalf("unable to open input: %v", err)
			}

		case 1:
			// No program, just the file path
			if source, err = os.Open(args[0]); err != nil {
				log.Fatalf("unable to open input: %v", err)
			}

		default:
			flag.Usage()
			os.Exit(1)
		}
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
		case ui.ActionInputBackspace:
			display.UpdateInputBackspace()
			display.RenderInput()

		case ui.ActionExit:
			display.Quit()
			os.Exit(0)

		case ui.ActionInput:
			display.UpdateInput()
			display.RenderInput()

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

		case ui.ActionSaveInput:
			display.UpdateSaveInput()
			display.RenderFilePrompt()

		case ui.ActionSavePrompt:
			// TODO Support cancellation in save prompt
			display.SaveInputMode = true
			if err := display.RenderFilePrompt(); err != nil {
				log.Fatalf("unable to open save form: %v", err)
			}

		case ui.ActionSavePromptBackspace:
			display.UpdateSaveInputBackspace()
			display.RenderFilePrompt()

		case ui.ActionSaveSubmit:
			display.Quit()
			// TODO handle "mkdir -p" style directory create
			f, err := os.OpenFile(
				fmt.Sprintf("%s/%s", cwd, display.SavePath),
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				0666,
			)
			if err != nil {
				log.Fatalf("could not open save file: %v", err)
			}
			out, err := processor.Process(bytes.NewReader(jsonData), display.Input)
			if err != nil {
				// TODO distinguish between normal parse errors and crashable errors
				if debugFile != nil {
					fmt.Fprintf(debugFile, "parse error: %v\n", err)
					fmt.Fprintf(debugFile, "program: %s\n", display.Input)
				}
			} else {
				_, err := io.Copy(f, out)
				if err != nil {
					log.Fatalf("could not write results to file: %v", err)
				}
				os.Exit(0)
			}

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
			// TODO need some kind of UI indicator to indicate active options
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
	}
}

func inputOnStdin(stdin *os.File) bool {
	stat, err := stdin.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to stat stdin: %v\n", err)
		// We'll assume reading from file, but we'll probably crash somewhere later.
		return true
	}
	return (stat.Mode() & os.ModeNamedPipe) != 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `jq-live: on-the-fly JSON processing

Usage: jq-live <program> [path]
`)
	flag.PrintDefaults()
}
