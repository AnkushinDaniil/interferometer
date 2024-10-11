package app

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/AnkushinDaniil/interferometer/entity"
)

type App struct {
	Source    string
	Output    string
	Params    *entity.Parameters
	filenames []string
	lines     []*entity.Line
}

func New(source, output string, params *entity.Parameters) *App {
	return &App{
		Source: source,
		Output: output,
		Params: params,
	}
}

func (a *App) Run(ctx context.Context) error {
	appTime := time.Now()
	defer func() {
		fmt.Printf("App finished in %v\n", time.Since(appTime))
	}()
	if err := a.getFilenames(); err != nil {
		return fmt.Errorf("failed to get filenames: %w", err)
	}

	a.lines = make([]*entity.Line, len(a.filenames))
	var err error
	for i, filename := range a.filenames {
		a.lines[i], err = entity.NewLine(
			filename, a.Params,
		)
		a.lines[i].SetVisibilityFromFile(filename)
		if err != nil {
			return fmt.Errorf("failed to get visibility data: %w", err)
		}
		fmt.Printf("Visibility data for %s is calculated\n", filename)
	}

	line := a.createChart()
	fmt.Println("Chart created")

	f, _ := os.Create(a.Output)
	defer f.Close()
	if err := line.Render(f); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}
	fmt.Println("Chart saved")

	return nil
}

func (a *App) getFilenames() error {
	dir, err := os.Open(a.Source)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer dir.Close()

	fileInfo, err := dir.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	a.filenames = make([]string, 0)

	if fileInfo.IsDir() {
		if filepath.Ext(fileInfo.Name()) == ".bin" {
			a.filenames = append(a.filenames, fileInfo.Name())
			fmt.Printf("Found file: %s\n", fileInfo.Name())
		}
	} else {
		files, err := dir.Readdir(0)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}
		for _, v := range files {
			if filepath.Ext(v.Name()) == ".bin" {
				a.filenames = append(a.filenames, v.Name())
				fmt.Printf("Found file: %s\n", v.Name())
			}
		}
	}

	if len(a.filenames) == 0 {
		return fmt.Errorf("no files found in directory")
	}

	return nil
}

func (a *App) createChart() *charts.Line {
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

	zeroIdx := 0
	x := make([]float64, len(a.lines[0].Data()))
	dx := a.Params.Length / float64(len(a.lines[0].Data()))
	for i := range a.lines[0].Data() {
		x[i] = float64(i-zeroIdx) * dx
	}
	line.SetXAxis(x)

	for i := range a.lines {
		line.AddSeries(fmt.Sprintf("Видность %d", i), a.lines[i].Data())
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
