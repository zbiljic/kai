package commit

import "regexp"

var commitMessageRegex = regexp.MustCompile(`^(?P<type>\w+)(\((?P<scope>[\w\-\.\/]+)\))?(!)?: (?P<message>.+)$`)

func ParseMessage(message string) Message {
	match := commitMessageRegex.FindStringSubmatch(message)
	if len(match) == 0 {
		return Message{
			CommitMessage: message,
		}
	}

	typeString := match[1]
	scopeString := match[3]
	breakingString := match[4]
	messageString := match[5]

	if typeString == "" {
		return Message{
			CommitMessage: message,
		}
	}

	return Message{
		Type:          typeString,
		Scope:         scopeString,
		Breaking:      breakingString != "",
		CommitMessage: messageString,
	}
}
