package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

// Termbox draws the jq-live UI via termbox
type Termbox struct {
	Debug      io.WriteCloser
	Input      string
	dirtyInput bool
	events     chan (Action)
	newInput   rune
}

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

// Action defines the events and interactions possible in the application
type Action uint8

// The following are the known Actions from the app to handle
const (
	ActionBackspace Action = iota
	ActionExit
	ActionInput
	ActionSubmit // TODO replace with live editing
)

// Events returns a channel of Actions that are sent through the application
func (t *Termbox) Events() chan (Action) {
	t.events = make(chan (Action))

	go func() {
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch key := ev.Key; key {
				case termbox.KeyEsc, termbox.KeyCtrlC, termbox.KeyCtrlD:
					t.events <- ActionExit
				case termbox.KeyEnter:
					t.events <- ActionSubmit
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					t.events <- ActionBackspace
				case termbox.KeySpace:
					t.newInput = ' '
					t.events <- ActionInput
				default:
					if ev.Ch != 0 {
						t.newInput = ev.Ch
						t.events <- ActionInput
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

	termbox.Flush()
	return err
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

	err = termbox.Flush()
	return err
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
