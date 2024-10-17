package format

import "fmt"

type Format int8

const (
	HTML Format = iota
	Png
	Csv
)

func UnmarshalText(text string) (Format, error) {
	switch text {
	case "html":
		return HTML, nil
	case "png":
		return Png, nil
	case "csv":
		return Csv, nil
	default:
		return 0, fmt.Errorf("invalid format: %q", text)
	}
}
