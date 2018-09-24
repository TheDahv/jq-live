// Package ui handles user input, JSON processing output, and sending
// interactions and commands to the JSON processor.
//
// The basic layout of the UI should include:
// - One-row input for entering jq programs
// - The rest of the container for displaying JSON processor output
//
// The data flow should flow through the following loop:
// - Initial state
// - Actions representing changes or events in the program
// - Functions that listen for a given action and call methods on a UI
//	 implementation to update the state
// - Render the new state into application UI
//
// This should seem familiar to web programmers familiar with the Flux/Redux
// flow. However, since Go doesn't have union types that carry data, and since
// we want tighter control over memory, we use internal fields and buffers to
// manage interim states and reuse memory.
package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

// TODO move exported fields behind methods so we can wrap in an interface.
// It will make porting to other libraries easier:
// https://github.com/gdamore/tcell

// Termbox draws the jq-live UI via termbox
type Termbox struct {
	Debug         io.WriteCloser
	Input         string
	SaveInputMode bool
	SavePath      string
	dirtyInput    bool
	events        chan (Action)
	newInput      rune
	flushLock     sync.Mutex
}

const filePrompt = "save to: "

// Start initializes the UI and returns a handle to the manager
func (t *Termbox) Start(initialProgram string) error {
	if err := termbox.Init(); err != nil {
		return fmt.Errorf("could not init termbox: %v", err)
	}
	termbox.SetCursor(0, 0)

	t.Input = initialProgram

	// First render
	t.dirtyInput = true

	return nil
}

// UpdateInput appends the new input character to the internal input buffer
func (t *Termbox) UpdateInput() {
	if t.newInput != 0 {
		t.Input = fmt.Sprintf("%s%s", t.Input, string(t.newInput))
		t.newInput = 0
		t.dirtyInput = true
	}
}

// UpdateInputBackspace removes the last character from the input
func (t *Termbox) UpdateInputBackspace() {
	if len(t.Input) == 0 {
		return
	}

	t.Input = t.Input[0 : len(t.Input)-1]
	t.dirtyInput = true
}

// UpdateSaveInput appends the new input character to the internal input buffer
func (t *Termbox) UpdateSaveInput() {
	if t.newInput != 0 {
		t.SavePath = fmt.Sprintf("%s%s", t.SavePath, string(t.newInput))
		t.newInput = 0
		t.dirtyInput = true
	}
}

// UpdateSaveInputBackspace removes the last character from the input
func (t *Termbox) UpdateSaveInputBackspace() {
	if len(t.SavePath) == 0 {
		return
	}

	t.SavePath = t.SavePath[0 : len(t.SavePath)-1]
	t.dirtyInput = true
}

// Action defines the events and interactions possible in the application
type Action uint8

// The following are the known Actions from the app to handle
const (
	ActionExit Action = iota
	ActionInput
	ActionInputBackspace
	ActionPrint
	ActionSaveInput
	ActionSavePrompt
	ActionSavePromptBackspace
	ActionSaveSubmit
	ActionSubmit
	ActionToggleCompact
	ActionToggleRaw
)

// Events returns a channel of Actions that are sent through the application
func (t *Termbox) Events() chan (Action) {
	t.events = make(chan (Action))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Termbox will occasionally crash reading new input into its event
				// buffers. No idea why, and I don't want to sync time into finding it.
				// Log and continue listening for events.
				//
				// https://github.com/nsf/termbox-go/issues/166
				// https://github.com/nsf/termbox-go/issues/169
				t.debugf("termbox events buffer failed: %v\n", r)
			}
		}()

		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch key := ev.Key; key {
				case termbox.KeyEsc, termbox.KeyCtrlC, termbox.KeyCtrlD:
					t.events <- ActionExit
				case termbox.KeyCtrlE:
					t.events <- ActionToggleCompact
				case termbox.KeyCtrlP:
					t.events <- ActionPrint
				case termbox.KeyCtrlR:
					t.events <- ActionToggleRaw
				case termbox.KeyCtrlS:
					t.events <- ActionSavePrompt
				case termbox.KeyEnter:
					if t.SaveInputMode {
						t.events <- ActionSaveSubmit
					} else {
						t.events <- ActionSubmit
					}
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					if t.SaveInputMode {
						t.events <- ActionSavePromptBackspace
					} else {
						t.events <- ActionInputBackspace
					}
				case termbox.KeySpace:
					t.newInput = ' '
					t.events <- ActionInput
				default:
					t.debugf("key pressed: %d. Mod: %v\n", ev.Ch, ev.Mod)
					if ev.Ch != 0 {
						t.newInput = ev.Ch
						if t.SaveInputMode {
							t.events <- ActionSaveInput
						} else {
							t.events <- ActionInput
						}
					}
				}
			}
		}
	}()

	return t.events
}

// RenderInput updates the input display to match the internal buffer
func (t *Termbox) RenderInput() error {
	t.debugf("input: %s\n", t.Input)

	scanner := bufio.NewScanner(strings.NewReader(t.Input))
	scanner.Split(bufio.ScanRunes)

	var x int
	for scanner.Scan() {
		r, w := utf8.DecodeRune(scanner.Bytes())
		termbox.SetCell(x, 0, r, termbox.ColorDefault, termbox.ColorDefault)

		x += w
	}

	// Clear rest of the input on the row
	w, _ := termbox.Size()
	for x := len(t.Input); x < w; x++ {
		termbox.SetCell(x, 0, 0, termbox.ColorDefault, termbox.ColorDefault)
	}

	termbox.SetCursor(len(t.Input), 0)
	err := scanner.Err()
	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return fmt.Errorf("could not process data for printing: %v", err)
	}
	return t.Flush()
}

// RenderFilePrompt switches the UI to the file input
func (t *Termbox) RenderFilePrompt() error {
	t.debugf("renderfileprompt: %v\n", t.SaveInputMode)
	if !t.SaveInputMode {
		return nil
	}

	// Clear input row
	w, _ := termbox.Size()
	for x := 0; x < w; x++ {
		termbox.SetCell(x, 0, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}

	t.debugf("rendering prompt: '%s'\n", filePrompt+t.SavePath)
	prompt := filePrompt + t.SavePath
	scanner := bufio.NewScanner(strings.NewReader(prompt))
	scanner.Split(bufio.ScanRunes)

	var x int
	for scanner.Scan() {
		r, w := utf8.DecodeRune(scanner.Bytes())
		t.debugf("printing %s\n", string(r))
		termbox.SetCell(x, 0, r, termbox.ColorDefault, termbox.ColorDefault)
		x += w
	}
	termbox.SetCursor(len(prompt), 0)

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("could not print save prompt: %v", err)
	}

	return t.Flush()
}

// RenderResults renders the results of jq output to the screen
func (t *Termbox) RenderResults(data io.Reader) error {
	var (
		xOffset   int
		yPos      = 1 // Offset for input row
		xPos      int
		xPosReset = xOffset + 0

		err error

		rows   *bufio.Reader
		row    []byte
		rowRdr *bufio.Reader
	)

	// Clean up buffer
	w, h := termbox.Size()
	for y := 1; y < h; y++ {
		for x := 0; x < w; x++ {
			termbox.SetCell(x, y, 0, termbox.ColorDefault, termbox.ColorDefault)
		}
	}

	rows = bufio.NewReader(data)
	for {
		xPos = xPosReset
		if row, err = rows.ReadBytes('\n'); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		rowRdr = bufio.NewReader(bytes.NewReader(row))
		for {
			var tok rune
			var width int
			if tok, width, err = rowRdr.ReadRune(); err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}

			termbox.SetCell(
				xPos,
				yPos,
				tok,
				termbox.ColorDefault,
				termbox.ColorDefault,
			)
			xPos += width
		}

		yPos++
	}

	if err == io.EOF {
		err = nil
	}

	return t.Flush()
}

// Flush prints any unprinted UI changes to the screen
//
// termbox-go Flush() is not goroutine safe, so we're protecting access if
// updates come in very quickly.
// https://github.com/nsf/termbox-go/issues/113
func (t *Termbox) Flush() error {
	t.flushLock.Lock()
	defer t.flushLock.Unlock()

	return termbox.Flush()
}

// Quit ends the program, gives the display back to the terminal, and performs
// any required cleanup
func (t *Termbox) Quit() {
	close(t.events)
	t.events = nil
	termbox.Close()
}

// debugf writes to the debug path, if it exists
func (t *Termbox) debugf(format string, args ...interface{}) {
	if t.Debug != nil {
		fmt.Fprintf(t.Debug, "[UI] "+format, args...)
	}
}
