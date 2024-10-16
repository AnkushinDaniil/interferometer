package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	log "github.com/sirupsen/logrus"

	"github.com/AnkushinDaniil/interferometer/entity"
)

type App struct {
	Source string
	Output string
	Params *entity.Parameters
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
		log.WithField("time", time.Since(appTime)).Debug("App finished")
	}()
	log.WithFields(log.Fields{
		"source": a.Source,
		"output": a.Output,
		"time":   a.Params.Time,
		"length": a.Params.Length,
		"speed":  a.Params.Speed,
		"deltaT": a.Params.DeltaT,
		"Lambda": a.Params.Lambda,
	}).Debug("App started")

	filePaths, err := getFilenames(a.Source)
	if err != nil {
		return fmt.Errorf("failed to get filenames: %w", err)
	}

	lines, err := linesFromFiles(filePaths, a.Params)
	if err != nil {
		return fmt.Errorf("failed to get lines from files: %w", err)
	}

	line := a.createChart(lines)
	log.Info("Chart created")

	f, err := os.Create(a.Output)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	renderTime := time.Now()
	if err := line.Render(f); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}
	log.WithField("time", time.Since(renderTime)).Info("Chart rendered and saved")

	return nil
}

func getFilenames(source string) ([]string, error) {
	dir, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory: %w", err)
	}
	defer dir.Close()

	fileInfo, err := dir.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	filePaths := make([]string, 0)

	if fileInfo.IsDir() {
		files, err := dir.Readdir(0)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}
		for _, fileInfo := range files {
			if filepath.Ext(fileInfo.Name()) == ".bin" {
				filePaths = append(filePaths, filepath.Join(source, fileInfo.Name()))
				log.WithField("name", fileInfo.Name()).Debug("Found file")
			}
		}
	} else if filepath.Ext(fileInfo.Name()) == ".bin" {
		filePaths = append(filePaths, source)
		log.WithField("name", fileInfo.Name()).Debug("Found file")
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files found in directory")
	}

	return filePaths, nil
}

func linesFromFiles(filePaths []string, params *entity.Parameters) ([]*entity.Line, error) {
	lines := make([]*entity.Line, 0, len(filePaths))
	for _, filePath := range filePaths {
		log.WithField("name", filePath).Debug("Creating line")
		line, err := entity.NewLine(
			strings.TrimSuffix(filePath, filepath.Ext(filePath)),
			params,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create line: %w", err)
		}
		log.Debug("Line created")
		lines = append(lines, line)
	}
	return lines, nil
}

func (a *App) createChart(lines []*entity.Line) *charts.Line {
	startTime := time.Now()
	defer func() {
		log.WithFields(log.Fields{
			"time":  time.Since(startTime),
			"lines": len(lines),
		}).Debug("Creating chart")
	}()
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

	x := make([]float64, len(lines[0].Data()))
	dx := a.Params.Length / float64(len(lines[0].Data()))
	zeroIdx := getMaxIdx(lines[0].Data())
	for i := range lines[0].Data() {
		x[i] = float64(i-zeroIdx) * dx
	}
	line.SetXAxis(x)

	for i := range lines {
		line.AddSeries(fmt.Sprintf("Видность %s", lines[i].Name()), lines[i].Data())
	}
	return line
}

func getMaxIdx(data []opts.LineData) int {
	maxIdx := 0
	for i, d := range data {
		if d.Value.(float64) > data[maxIdx].Value.(float64) {
			maxIdx = i
		}
	}
	return maxIdx
}
