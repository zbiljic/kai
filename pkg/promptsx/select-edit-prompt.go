package promptsx

import (
	"os"

	"github.com/orochaa/go-clack/core"
	"github.com/orochaa/go-clack/core/utils"
	"github.com/orochaa/go-clack/core/validator"
)

type EditableValue[TValue any] struct {
	Value TValue
	Edit  bool
}

type SelectEditOption[TValue any] struct {
	Label  string
	Value  TValue
	Key    string
	IsEdit bool
}

type SelectEditPrompt[TValue any] struct {
	core.Prompt[EditableValue[TValue]]
	Options []*SelectEditOption[TValue]
	EditKey core.KeyName
}

type SelectEditPromptParams[TValue any] struct {
	Input   *os.File
	Output  *os.File
	Options []*SelectEditOption[TValue]
	EditKey core.KeyName
	Render  func(p *SelectEditPrompt[TValue]) string
}

func NewSelectEditPrompt[TValue any](params SelectEditPromptParams[TValue]) *SelectEditPrompt[TValue] {
	v := validator.NewValidator("SelectEditPrompt")
	v.ValidateRender(params.Render)
	v.ValidateOptions(len(params.Options))

	startIndex := 0

	for _, option := range params.Options {
		if value, ok := any(option.Value).(string); ok && value == "" {
			// option.Value = any(option.Key).(TValue)
			option.Value = any(option.Label).(TValue)
		}
	}

	if params.EditKey == "" {
		params.EditKey = "e"
	}

	var p SelectEditPrompt[TValue]
	p = SelectEditPrompt[TValue]{
		Prompt: *core.NewPrompt(core.PromptParams[EditableValue[TValue]]{
			Input:  params.Input,
			Output: params.Output,
			// InitialValue: params.Options[startIndex].Value,
			InitialValue: EditableValue[TValue]{
				Value: params.Options[startIndex].Value,
			},
			CursorIndex: startIndex,
			Render:      core.WrapRender[EditableValue[TValue]](&p, params.Render),
		}),
		Options: params.Options,
		EditKey: params.EditKey,
	}

	p.On(core.KeyEvent, func(args ...any) {
		p.handleKeyPress(args[0].(*core.Key))
	})

	return &p
}

func (p *SelectEditPrompt[TValue]) handleKeyPress(key *core.Key) {
	for i, option := range p.Options {
		if key.Name == core.KeyName(option.Key) {
			p.State = core.SubmitState
			// p.Value = option.Value
			p.Value = EditableValue[TValue]{
				Value: option.Value,
				Edit:  false,
			}
			p.CursorIndex = i
			return
		}
	}

	switch key.Name {
	case p.EditKey:
		for i, option := range p.Options {
			if i == p.CursorIndex {
				p.State = core.SubmitState
				// p.Value = option.Value
				p.Value = EditableValue[TValue]{
					Value: option.Value,
					Edit:  true,
				}
				return
			}
		}
	case core.UpKey, core.LeftKey:
		p.CursorIndex = utils.MinMaxIndex(p.CursorIndex-1, len(p.Options))
	case core.DownKey, core.RightKey:
		p.CursorIndex = utils.MinMaxIndex(p.CursorIndex+1, len(p.Options))
	case core.HomeKey:
		p.CursorIndex = 0
	case core.EndKey:
		p.CursorIndex = len(p.Options) - 1
	case core.EnterKey, core.CancelKey:
	default:
		break
	}

	if p.CursorIndex >= 0 && p.CursorIndex < len(p.Options) {
		// p.Value = p.Options[p.CursorIndex].Value
		p.Value = EditableValue[TValue]{
			Value: p.Options[p.CursorIndex].Value,
		}
	}
}
