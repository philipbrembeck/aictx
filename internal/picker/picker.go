package picker

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

// IsTerminal returns true if stdin/stdout are a terminal.
func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stdin.Fd())
}

// Pick shows an interactive list picker with arrow keys + enter.
// Returns the selected item or empty string if cancelled (Esc / Ctrl-C).
func Pick(items []string, current string) (string, error) {
	if len(items) == 0 {
		return "", nil
	}

	// Start with current item selected
	cursor := 0
	for i, item := range items {
		if item == current {
			cursor = i
			break
		}
	}

	// Switch to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("enabling raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	render(items, cursor, current)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}

		switch {
		case n == 1 && buf[0] == 13: // Enter
			clearList(len(items))
			return items[cursor], nil
		case n == 1 && (buf[0] == 3 || buf[0] == 27): // Ctrl-C or bare Esc
			// For bare Esc, need to check it's not an arrow sequence
			if buf[0] == 27 {
				// Could be start of escape sequence; handled below if n>1
				// For n==1 it's a bare Esc
			}
			clearList(len(items))
			return "", nil
		case n == 1 && buf[0] == 'q': // q to quit
			clearList(len(items))
			return "", nil
		case n == 1 && buf[0] == 'k': // vim up
			if cursor > 0 {
				cursor--
			}
		case n == 1 && buf[0] == 'j': // vim down
			if cursor < len(items)-1 {
				cursor++
			}
		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			switch buf[2] {
			case 65: // Up
				if cursor > 0 {
					cursor--
				}
			case 66: // Down
				if cursor < len(items)-1 {
					cursor++
				}
			}
		default:
			continue
		}

		// Move cursor back to start of list and re-render
		fmt.Printf("\033[%dA", len(items))
		render(items, cursor, current)
	}
}

func render(items []string, cursor int, current string) {
	for i, item := range items {
		marker := "  "
		if item == current {
			marker = "* "
		}

		if i == cursor {
			// Highlighted: bold + reverse
			fmt.Printf("\033[1;7m%s%s\033[0m\033[K\r\n", marker, item)
		} else {
			fmt.Printf("%s%s\033[K\r\n", marker, item)
		}
	}
}

func clearList(n int) {
	fmt.Printf("\033[%dA", n)
	for i := 0; i < n; i++ {
		fmt.Print("\033[K\r\n")
	}
	fmt.Printf("\033[%dA", n)
}
