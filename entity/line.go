package entity

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

type Line struct {
	name    string
	data    []opts.LineData
	minimum int32
	params  *Parameters
	winSize int
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
		fmt.Printf("Visibility data for %s is calculated in %v\n", filename, time.Since(timestamp))
	}()

	l.name = strings.TrimSuffix(filename, filepath.Ext(filename))

	var err error
	l.minimum, err = findMinimumInFile(filename)
	if err != nil {
		return fmt.Errorf("failed to find minimum in file: %w", err)
	}
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

	br := bufio.NewReader(file)

	valueChan := make(chan int32, 1<<10)
	visibilityChan := make(chan float64, 1<<10)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer close(valueChan)
		defer wg.Done()
		readValues(br, valueChan)
	}()

	wg.Add(1)
	go func() {
		defer close(visibilityChan)
		defer wg.Done()
		l.calculateVisibility(valueChan, visibilityChan)
	}()

	dx := l.params.Speed * l.params.DeltaT
	l.winSize = int(l.params.Lambda/dx) * l.params.PeriodNumber

	l.data = make([]opts.LineData, 0, (int(signalLength)+l.winSize)/l.winSize)
	for visibilityValue := range visibilityChan {
		l.data = append(l.data, opts.LineData{Value: visibilityValue})
	}
	wg.Wait()
	return nil
}

func findMinimumInFile(filename string) (int32, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	br := bufio.NewReader(file)
	values := [8]byte{}
	minimum := int32(0)
	for {
		_, err := io.ReadFull(br, values[:])
		if err != nil {
			if err != io.EOF {
				return 0, fmt.Errorf("failed to read values: %w", err)
			}
			break
		}
		value := int32(binary.BigEndian.Uint64(values[:]))
		if value < minimum {
			minimum = value
		}
	}
	return minimum, nil
}

func readValues(br *bufio.Reader, valueChan chan<- int32) {
	values := [8]byte{}
	for {
		_, err := io.ReadFull(br, values[:])
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}
		valueChan <- int32(binary.BigEndian.Uint64(values[:]))
	}
}

func (l *Line) calculateVisibility(valueChan <-chan int32, visibilityChan chan<- float64) {
	i := 0
	j := 0
	maxVisibility := float64(0)
	minimum, maximum := int32(math.MaxInt32), int32(0)
	for value := range valueChan {
		value -= l.minimum
		if value < minimum {
			minimum = value
		}
		if value > maximum {
			maximum = value
		}
		i++
		if i == l.winSize {
			visibilityValue := float64(maximum-minimum) / float64(maximum+minimum)
			if visibilityValue > maxVisibility {
				maxVisibility = visibilityValue
				l.zeroIdx = j
			}
			visibilityChan <- visibilityValue

			i = 0
			minimum, maximum = math.MaxInt32, int32(0)
		}
	}
}

func (l *Line) GetZeroIdx() int {
	return l.zeroIdx
}
