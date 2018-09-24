# jq-live

[`jq`](https://stedolan.github.io/jq/) is a fantastic tool for working with
JSON on the command line, but `jq-live` makes it easier to explore large
payloads with which you're not already familiar.

Add it to your data processing chain when you would normally use `jq` but don't
know what selectors or processors you need yet.

    $ jq-live -h
    jq-live: on-the-fly JSON processing

    Usage: jq-live <program> [path]
      -c    compact output
      -debug string
            path to write debug information
      -r    raw output

    keyboard shortcuts:

      Ctrl-E: Toggle expanded or compact results
      Ctrl-R: Toggle raw results
      Ctrl-P: Print current results to stdout and quit
      Ctrl-S: Save current results to specified file and quit

Call `jq-live` in one of a few ways:

- Call with a path to a file
  - `jq-live something.json`
  - `jq-live 'keys' something.json`
- Pass JSON through `stdin`
  - `cat $JSON | jq-live`
  - `jq-live 'keys' < something.json`
  - `curl -s http://something.com/endpoint.json | jq-live | wc -l`

Here's some `jq-live` in action:

[![jq-live demo asciicast](https://asciinema.org/a/o6KI0knzMIoILEqKZyDv6pHR1.png)](https://asciinema.org/a/o6KI0knzMIoILEqKZyDv6pHR1)

## Getting `jq-live`

- Binary downloads: _Not available yet_
- Package managers: _Not available yet_
- Via `go get`
  - Run `go get -u gitlab.com/thedahv/jq-live`

## Possible Future Features or Changes

Features:
- In-app keyboard shortcuts display

JSON Processing:
- Implement `jq` as library calls rather than shell callouts

UI Features:
- [Readline](https://tiswww.case.edu/php/chet/readline/rltop.html) motions support
- Explore [gdamore/tcell](https://github.com/gdamore/tcell) as a termbox
  alternative

## Known Issues

- `jq` required to be installed on host
- Only tested on OSX and Linux
- Occasional input event crashing from Termbox
  [[issue]](https://github.com/nsf/termbox-go/issues/166)

Code TODOs:

`main.go`

- Support cancellation in save prompt
- handle "mkdir -p" style directory create
- need some kind of UI indicator to indicate active options
- distinguish between normal parse errors and crashable errors

`ui/ui.go`

- move exported fields behind methods so we can wrap in an interface

## Development

To get started working on `jq-live`, download the project to your computer with:

    go get -u gitlab.com/thedahv/jq-live

This pulls down the code, its dependencies, and installs a copy of `jq-live` in
`$GOPATH/bin`.

### Modules

This project is also a chance for me to play with Go modules. Go modules require
at least Go 1.11. If you have a satisfactory Go installation, activate module
mode:

    export GO111MODULE=on

In either scenario, run `go get -u gitlab.com/thedahv/jq-live` to get package
dependencies.

### JSON Processing

The `json` package manages sending data and program input into and out of the jq
implementation.

The initial implementation relies on the `jq` binary being installed on the host
to bootstrap that functionality.

Plans include re-implementing `jq` in Go to avoid this dependency.

### UI

The UI is currently managed by [termbox-go](https://github.com/nsf/termbox-go).
Termbox takes control over the current terminal for the lifespan of the command,
draws the state of the application in
[Cells](https://godoc.org/github.com/nsf/termbox-go#Cell) arranged in rows and
columns. Use [`SetCell`](https://godoc.org/github.com/nsf/termbox-go#SetCell) to
write a character to a cell.

Please read the package documentation in `ui/ui.go` to learn about data flow and
management.
