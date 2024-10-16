package entity

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
	log "github.com/sirupsen/logrus"
)

type Line struct {
	name    string
	data    []opts.LineData
	params  *Parameters
	zeroIdx int
}

func NewLine(name string, params *Parameters) (*Line, error) {
	if name == "" {
		return nil, errors.New("name is empty")
	}
	return &Line{name: name, params: params}, nil
}

func (l *Line) Name() string {
	return l.name
}

func (l *Line) Data() []opts.LineData {
	return l.data
}

func (l *Line) SetVisibilityFromFile(filename string) error {
	timestamp := time.Now()
	defer func() {
		log.WithFields(log.Fields{
			"name": l.name,
			"time": time.Since(timestamp),
		}).Debug("Visibility data is calculated")
	}()

	dx := l.params.Speed * l.params.DeltaT
	winSize := int(l.params.Lambda/dx) * l.params.PeriodNumber
	log.WithField("winSize", winSize).Debug("Window size is calculated")

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	signalLength := fi.Size() / 8
	log.WithField("signalLength", signalLength).Debug("Signal length is calculated")

	visibilityLength := (int(signalLength) + winSize - 1) / winSize
	log.WithField("visibilityLength", visibilityLength).Debug("Visibility length is calculated")

	minMax := make([][2]int32, 0, visibilityLength)
	br := bufio.NewReader(file)

	valueChan := make(chan int32, 1024)
	minMaxChan := make(chan [2]int32, 1024)

	go func() {
		if err := readValues(br, valueChan); err != nil {
			log.WithError(err).Error("Error reading values")
		}
	}()

	go l.calculateMinMax(valueChan, minMaxChan, winSize)

	minValue := int32(math.MaxInt32)
	for minMaxValue := range minMaxChan {
		minMax = append(minMax, minMaxValue)
		if minMaxValue[0] < minValue {
			minValue = minMaxValue[0]
		}
	}

	log.WithField("minValue", minValue).Debug("Min value is calculated")

	// Adjust min-max values in bulk to improve efficiency
	for i := range minMax {
		minMax[i][0] -= minValue
		minMax[i][1] -= minValue
	}

	log.WithField("minMax length", len(minMax)).Debug("Min and max values are calculated")

	l.data = make([]opts.LineData, 0, visibilityLength)
	for _, mm := range minMax {
		diff := float64(mm[1]) - float64(mm[0])
		sum := float64(mm[1]) + float64(mm[0])

		visibility := 0.0
		if sum != 0 {
			visibility = diff / sum
		}

		l.data = append(l.data, opts.LineData{
			Value: visibility,
		})
	}

	return nil
}

func readValues(br *bufio.Reader, valueChan chan<- int32) error {
	defer close(valueChan)
	log.Debug("Reading values")
	values := [8]byte{}
	for {
		_, err := io.ReadFull(br, values[:])
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read values: %w", err)
		}
		valueChan <- int32(binary.BigEndian.Uint64(values[:]))
	}
	return nil
}

func (l *Line) calculateMinMax(
	valueChan <-chan int32,
	minMaxChan chan<- [2]int32,
	winSize int,
) {
	defer close(minMaxChan)
	log.Debug("Calculating min and max values")
	i := 0
	winMin, winMax := int32(math.MaxInt32), int32(math.MinInt32)
	for value := range valueChan {
		if value < winMin {
			winMin = value
		}
		if value > winMax {
			winMax = value
		}
		i++
		if i == winSize {
			minMaxChan <- [2]int32{winMin, winMax}
			winMin, winMax = int32(math.MaxInt32), int32(math.MinInt32)
			i = 0
		}
	}
}
