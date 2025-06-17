package promptsx

import (
	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/prompts/symbols"
	"github.com/orochaa/go-clack/third_party/picocolors"
)

// Note displays a formatted note box with a title, message, and borders.
func Note(msg string) {
	prompts.Note(msg, prompts.NoteOptions{})
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
