package main

import (
	"bufio"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

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

	file, err := os.Open("stream_20240924-110038.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	br := bufio.NewReader(file)
	values := [8]byte{}
	interference := make([]int32, 100000)
	for i := range 100000 {
		_, err := io.ReadFull(br, values[:])
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}
		interference[i] = int32(binary.BigEndian.Uint64(values[:]))
	}
	line := createChart([][]int32{interference})
	f, _ := os.Create("Interference.html")
	defer f.Close()
	line.Render(f)
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
			Name: "м",
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(true),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:  "Интерференция, усл. ед.",
			Type:  "value",
			Show:  opts.Bool(true),
			Scale: opts.Bool(true),
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(true),
			},
		}),
	)

	// fmt.Println(zeroIdx)
	x := make([]float64, len(data[0]))
	for i := range data[0] {
		x[i] = float64(i) * DeltaX
	}
	line.SetXAxis(x)

	lineSeries := make([][]opts.LineData, len(data))
	for i := range data {
		lineSeries[i] = make([]opts.LineData, len(data[i]))
		for j, v := range data[i] {
			lineSeries[i][j] = opts.LineData{Value: v}
		}
		line.AddSeries(fmt.Sprintf("Интерференция %d", i), lineSeries[i])
	}

	return line
}
