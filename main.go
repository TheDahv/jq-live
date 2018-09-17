package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"gitlab.com/thedahv/jq-live/json"
)

func main() {
	var compact = flag.Bool("c", false, "compact output")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	var source io.Reader
	if len(args) == 1 {
		source = os.Stdin
	} else {
		var err error
		if source, err = os.Open(args[1]); err != nil {
			log.Fatalf("unable to open input: %v", err)
		}
	}

	processor, err := json.NewShell(json.OptionCompact(*compact))
	if err != nil {
		log.Fatalf("unable to start up processor: %v", err)
	}

	out, err := processor.Process(source, args[0])
	if err != nil {
		log.Fatalf("unable to run command: %v", err)
	}

	io.Copy(os.Stdout, out)
}

func usage() {
	fmt.Fprintf(os.Stderr, `jq-live: on-the-fly JSON processing

Usage: jq-live <program> [path]
`)
	flag.PrintDefaults()
}
