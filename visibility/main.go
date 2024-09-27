package main

import (
	"bufio"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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
	PeriodNumber = 30
	WinSize      = int(Lambda/DeltaX) * PeriodNumber
)

func main() {
	fmt.Println("Time", Time)
	fmt.Println("DataLength", DataLength)
	fmt.Println("Speed", Speed)
	fmt.Println("Length", Length)
	fmt.Println("DeltaX", DeltaX)
	fmt.Println("Lambda", Lambda)
	fmt.Println("PeriodNumber", PeriodNumber)
	fmt.Println("WinSize", WinSize)

	filenames := make([]string, 0)

	dir, err := os.Open(".")
	if err != nil {
		fmt.Println(err)
		return
	}
	files, err := dir.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range files {
		if filepath.Ext(v.Name()) == ".bin" {
			filenames = append(filenames, v.Name())
			fmt.Printf("Found file: %s\n", v.Name())
		}
	}

	visibilities := make([][]float64, len(filenames))
	for i, filename := range filenames {
		visibilities[i] = getVisibilityData(filename)
		fmt.Printf("Visibility data for %s is calculated\n", filename)
	}

	line := createChart(visibilities)
	fmt.Println("Chart created")

	f, _ := os.Create("Visibility.html")
	defer f.Close()
	line.Render(f)
	fmt.Println("Chart saved")
}

func getVisibilityData(filename string) []float64 {
	minimum, err := findMinimumInFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	br := bufio.NewReader(file)

	valueChan := make(chan int32, 1<<10)
	visibilityChan := make(chan float64, 1<<10)
	values := [8]byte{}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer close(valueChan)
		defer wg.Done()
		readValues(br, values, valueChan)
	}()

	wg.Add(1)
	go func() {
		defer close(visibilityChan)
		defer wg.Done()
		calculateVisibility(valueChan, minimum, visibilityChan)
	}()

	visibility := make([]float64, 0, int(DataLength)/WinSize+1)
	for visibilityValue := range visibilityChan {
		visibility = append(visibility, visibilityValue)
	}
	wg.Wait()
	return visibility
}

func readValues(br *bufio.Reader, values [8]byte, valueChan chan int32) {
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

func calculateVisibility(valueChan chan int32, minimum int32, visibilityChan chan float64) {
	values := make([]int32, WinSize)
	i := 0
	for value := range valueChan {
		values[i] = value - minimum
		i++
		if i == WinSize {
			min, max := minMaxSlice(values)
			visibilityChan <- float64(max-min) / float64(max+min)
			i = 0
		}
	}
}

func findMinimumInFile(filename string) (int32, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	br := bufio.NewReader(file)
	values := [8]byte{}
	minimum := int32(0)
	for {
		_, err := io.ReadFull(br, values[:])
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
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

func minMaxSlice[T cmp.Ordered](arr []T) (min, max T) {
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
			BackgroundColor: "#ffffff",
			Width:           "100%",
			Height:          "600px",
			PageTitle:       "Study of accuracy characteristics of the Michelson scanning interferometer",
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

	// zeroIdx := findIdxOfMax(data[0])
	zeroIdx := 76748 / PeriodNumber
	// fmt.Println(zeroIdx)
	x := make([]float64, len(data[0]))
	dx := Length / float64(len(data[0]))
	for i := range data[0] {
		x[i] = float64(i-zeroIdx) * dx
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
