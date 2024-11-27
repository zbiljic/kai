package promptsx

import (
	"fmt"

	"github.com/Mist3rBru/go-clack/core"
	"github.com/Mist3rBru/go-clack/core/validator"
	"github.com/Mist3rBru/go-clack/prompts/symbols"
	"github.com/Mist3rBru/go-clack/prompts/theme"
	"github.com/Mist3rBru/go-clack/third_party/picocolors"
)

type SelectEditParams[TValue comparable] struct {
	Message  string
	Options  []SelectEditOption[TValue]
	EditKey  core.KeyName
	EditHint string
}

func SelectEdit[TValue comparable](params SelectEditParams[TValue]) (EditableValue[TValue], error) {
	v := validator.NewValidator("SelectEdit")
	v.ValidateOptions(len(params.Options))

	var options []*SelectEditOption[TValue]
	for _, option := range params.Options {
		options = append(options, &SelectEditOption[TValue]{
			Label: option.Label,
			Value: option.Value,
			Key:   option.Key,
		})
	}

	p := NewSelectEditPrompt(SelectEditPromptParams[TValue]{
		Options: options,
		Render: func(p *SelectEditPrompt[TValue]) string {
			var value string

			switch p.State {
			case core.SubmitState, core.CancelState:
				if p.CursorIndex >= 0 && p.CursorIndex < len(p.Options) {
					value = p.Options[p.CursorIndex].Label
				}
			default:
				availableOptions := make([]string, len(p.Options))

				for _, option := range params.Options {
					for i, _option := range p.Options {
						if option.Label != _option.Label {
							continue
						}

						if i == p.CursorIndex {
							radio := picocolors.Green(symbols.RADIO_ACTIVE)
							label := option.Label
							key := picocolors.Cyan("[" + option.Key + "]")
							if params.EditHint != "" {
								hint := picocolors.Gray("(" + params.EditHint + ")")
								availableOptions[i] = fmt.Sprintf("%s %s %s %s", radio, key, label, hint)
							} else {
								availableOptions[i] = fmt.Sprintf("%s %s %s", radio, key, label)
							}
						} else {
							radio := picocolors.Dim(symbols.RADIO_INACTIVE)
							label := picocolors.Dim(option.Label)
							key := picocolors.Dim(picocolors.Cyan("[" + option.Key + "]"))
							availableOptions[i] = fmt.Sprintf("%s %s %s", radio, key, label)
						}

						break
					}
				}

				value = p.LimitLines(availableOptions, 3)
			}

			return theme.ApplyTheme(theme.ThemeParams[EditableValue[TValue]]{
				Ctx:             p.Prompt,
				Message:         params.Message,
				Value:           params.Options[p.CursorIndex].Label,
				ValueWithCursor: value,
			})
		},
	})

	return p.Run()
}
