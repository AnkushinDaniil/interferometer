package parameters

import (
	"github.com/AnkushinDaniil/interferometer/entity/format"
	"github.com/AnkushinDaniil/interferometer/entity/mode"
)

type Parameters struct {
	Mode         mode.Mode
	Format       format.Format
	Time         float64
	DeltaT       float64
	Speed        float64
	Length       float64
	Lambda       float64
	PeriodNumber int
}
