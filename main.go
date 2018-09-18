package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"gitlab.com/thedahv/jq-live/json"
	"gitlab.com/thedahv/jq-live/ui"
)

func main() {
	var compact = flag.Bool("c", false, "compact output")
	var debug = flag.String("debug", "", "path to write debug information")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	var (
		program string
		source  io.Reader
	)

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

	processor, err := json.NewShell(json.OptionCompact(*compact))
	if err != nil {
		log.Fatalf("unable to start up processor: %v", err)
	}

	out, err := processor.Process(source, program)
	if err != nil {
		log.Fatalf("unable to run command: %v", err)
	}

	display := &ui.Termbox{}
	if *debug != "" {
		display.Debug, err = os.OpenFile(
			*debug,
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			0666,
		)
		if err != nil {
			log.Fatalf("unable to open debug file: %v", err)
		}
	}
	fmt.Println("Starting with program:", program)
	if err := display.Start(program); err != nil {
		log.Fatalf("cannot start up display: %v", err)
	}

	// Initial draw
	display.RenderInput()
	if err := display.RenderResults(out); err != nil {
		log.Fatalf("cannot render result: %v", err)
	}

	for {
		switch action := <-display.Events(); action {
		case ui.ActionExit:
			display.Quit()
			os.Exit(0)
		case ui.ActionInput:
			display.UpdateInput()
		case ui.ActionBackspace:
			display.UpdateInputBackspace()
		}

		// TODO: flag dirty?
		display.RenderInput()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `jq-live: on-the-fly JSON processing

Usage: jq-live <program> [path]
`)
	flag.PrintDefaults()
}
