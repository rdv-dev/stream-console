package Types

type ConsoleType int

const (
	ConsoleTypeAll ConsoleType = iota
)

func (c ConsoleType) String() string {
	switch c {
	case ConsoleTypeAll:
		return "All"
	}

	return "invalid"
}

type SystemModule int

const (
	SystemMain SystemModule = iota
	SystemTwitchManager
)

func (s SystemModule) String() string {
	switch s {
	case SystemMain:
		return "Main"
	case SystemTwitchManager:
		return "TwitchManager"
	}

	return "invalid"
}

type SystemCommand struct {
	Command string
	Source  SystemModule
	Target  SystemModule
}

type ConsoleMessage struct {
	Message string
	Source  ConsoleType
}

func (mm *ConsoleMessage) PrintMessage() string {
	return mm.Message
}
