package promptsx

import (
	"fmt"
	"os"

	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/prompts/symbols"
	"github.com/orochaa/go-clack/third_party/picocolors"
)

// Note displays a formatted note box with a title, message, and borders.
func Note(msg string) {
	prompts.Note(msg, prompts.NoteOptions{})
}

// InfoNoSplitLines displays an informational message with a blue info symbol
// and no line splitting of the message.
func InfoNoSplitLines(msg string) {
	bar := picocolors.Gray(symbols.BAR)
	fmt.Fprintf(os.Stdout, "%s\r\n%s %s\r\n", bar, picocolors.Blue(symbols.INFO), msg)
}

// InfoWithLastLine displays an informational message with a blue info symbol
// and a last line.
func InfoWithLastLine(msg string) {
	prompts.Message(msg, prompts.MessageOptions{
		FirstLine: prompts.MessageLineOptions{
			Start: picocolors.Blue(symbols.INFO),
		},
		NewLine: prompts.MessageLineOptions{
			Start: picocolors.Gray(symbols.BAR),
		},
		LastLine: prompts.MessageLineOptions{
			Start: picocolors.Gray(symbols.BAR),
		},
	})
}

// ErrorNoSplitLines displays an error message with a red error symbol
// and no line splitting of the message.
func ErrorNoSplitLines(msg string) {
	bar := picocolors.Gray(symbols.BAR)
	fmt.Fprintf(os.Stdout, "%s\r\n%s %s\r\n", bar, picocolors.Red(symbols.ERROR), msg)
}
