package main

import (
	"bufio"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var (
	Time         = 80.0                // seconds
	DeltaT       = 1e-6                // seconds
	DataLength   = Time / DeltaT       // number of samples
	Speed        = 0.008               // m/s
	Length       = Time * Speed        // meters
	DeltaX       = Length / DataLength // meters
	Lambda       = 1550e-9             // meters
	PeriodNumber = 1
	WinSize      = int(Lambda/DeltaX) * PeriodNumber
)

func main() {
	file, err := os.Open("stream_20240913-141148.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	br := bufio.NewReader(file)

	valueChan := make(chan int32, 1<<10)
	resultChan := make(chan [2]int32, 1<<10)
	values := [8]byte{}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer close(valueChan)
		defer wg.Done()
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
	}()

	wg.Add(1)
	go func() {
		defer close(resultChan)
		defer wg.Done()
		values := make([]int32, WinSize)
		i := 0
		for value := range valueChan {
			values[i] = value
			i++
			if i == WinSize {
				min, max := minMax(values)
				resultChan <- [2]int32{min, max}
				i = 0
			}
		}
	}()

	maxes := make([]float32, 0, int(DataLength)/WinSize)
	mins := make([]float32, 0, int(DataLength)/WinSize)
	minimum := float32(0)

	for result := range resultChan {
		mins = append(mins, float32(result[0]))
		maxes = append(maxes, float32(result[1]))
		if float32(result[0]) < minimum {
			minimum = float32(result[0])
		}
	}
	wg.Wait()

	for i := range maxes {
		maxes[i] -= minimum
		mins[i] -= minimum
	}

	visibility := make([]float32, len(maxes))

	for i := range maxes {
		visibility[i] = (maxes[i] - mins[i]) / (maxes[i] + mins[i])
	}

	line := createChart([][]float32{visibility})

	f, _ := os.Create("Visibility.html")
	defer f.Close()
	line.Render(f)
}

func minMax(arr []int32) (min, max int32) {
	min, max = arr[0], arr[0]
	for _, v := range arr[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func createChart[T cmp.Ordered](data [][]T) *charts.Line {
	line := charts.NewLine()

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:     "100%",
			Height:    "600px",
			PageTitle: "Study of accuracy characteristics of the Michelson scanning interferometer",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "slider",
			Start:      0,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      0,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithLegendOpts(opts.Legend{
			Orient:       "horizontal",
			Show:         opts.Bool(true),
			SelectedMode: "multiple",
			Type:         "scroll",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
			AxisPointer: &opts.AxisPointer{
				Type: "cross",
				Snap: opts.Bool(true),
			},
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: opts.Bool(true),
			Top:  "0%",
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Type:  "png",
					Name:  "chart",
					Title: "Save as image",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:       opts.Bool(true),
					YAxisIndex: "default",
					Title: map[string]string{
						"zoom": "area zooming",
						"back": "restore area zooming",
					},
				},
				DataView: &opts.ToolBoxFeatureDataView{
					Show:  opts.Bool(true),
					Title: "Data view",
					Lang:  []string{"data view", "turn off", "refresh"},
				},
				Restore: &opts.ToolBoxFeatureRestore{
					Show:  opts.Bool(true),
					Title: "refresh",
				},
			},
		}),
		// AXIS
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Разность хода, м",
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(true),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:  "Видность",
			Type:  "value",
			Show:  opts.Bool(true),
			Scale: opts.Bool(true),
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(true),
			},
		}),
	)

	zeroIdx := findIdxOfMax(data[0])
	x := make([]float64, len(data[0]))
	for i := range data[0] {
		x[i] = float64(i-zeroIdx) * DeltaX
	}
	line.SetXAxis(x)

	lineSeries := make([][]opts.LineData, len(data))
	for i := range data {
		lineSeries[i] = make([]opts.LineData, len(data[i]))
		for j, v := range data[i] {
			lineSeries[i][j] = opts.LineData{Value: v}
		}
		line.AddSeries(fmt.Sprintf("Видность %d", i), lineSeries[i])
	}

	return line
}

func findIdxOfMax[T cmp.Ordered](data []T) int {
	maxIdx := 0
	for i, v := range data {
		if v > data[maxIdx] {
			maxIdx = i
		}
	}
	return maxIdx
}
