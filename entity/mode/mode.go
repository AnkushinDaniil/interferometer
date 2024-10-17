package mode

import "fmt"

type Mode uint8

const (
	Visibility Mode = iota
	Interference
)

func UnmarshalText(text string) (Mode, error) {
	switch text {
	case "v":
		return Visibility, nil
	case "i":
		return Interference, nil
	default:
		return 0, fmt.Errorf("invalid mode: %q", text)
	}
}
