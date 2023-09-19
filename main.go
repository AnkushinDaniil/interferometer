package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/wcharczuk/go-chart/v2"
	"io"
	"log"
	"math"
	"os"
	"slices"
	"sort"
	"strings"
)

const (
	htmlFilePath         = "charts.html"
	pngFilePath          = "charts.png"
	TIME                 = 260
	SPEED                = 1
	LENGTH       float64 = TIME * SPEED
	DIR                  = "C:\\Users\\User\\disser\\Coherence function 14.09.2023\\azamat_19.09.2023\\"
)

type line struct {
	yData  []float64
	kind   string
	name   string
	source string
	index  uint8
	dx     float64
}

func Float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func readData(dir string) []byte {
	data8, err := os.ReadFile(dir)
	if err != nil {
		log.Fatal(err)
	}
	return data8
}

func main() {
	files, err := os.ReadDir(DIR)
	if err != nil {
		log.Fatal(err)
	}

	x, lines := prepareData(files)

	//calculateCorrelationCoefficients(lines)
	//powers := getLines("dummy", "power", lines)
	//dummyVisibility := calculateVisibility(powers)
	//lines = append(lines, line{
	//	yData:  dummyVisibility,
	//	kind:   "visibility",
	//	name:   "calculated",
	//	source: "dummy",
	//	index:  0,
	//	dx:     x[1] - x[0],
	//})

	myChart := plotChart(x, lines)

	pageErr := createHtml(htmlFilePath, myChart)
	if pageErr != nil {
		log.Fatal(pageErr)
	}

	//png(lines, pngFilePath)

}

func png(lines []line, path string) {
	i1 := int(float64(len(lines[0].yData)) * 0.57)
	i2 := int(float64(len(lines[0].yData)) * 0.575)
	x := make([]float64, len(lines[0].yData))
	for i := 0; i < cap(x); i++ {
		x[i] = float64(i) / float64(cap(x))
	}
	graph := chart.Chart{
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name:    "name1",
				YValues: lines[0].yData[i1:i2],
				XValues: x[i1:i2],
			},
			chart.ContinuousSeries{
				Name:    "name2",
				YValues: lines[0].yData[i1:i2],
				XValues: x[i1:i2],
			},
		},
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
	}
	err = graph.Render(chart.PNG, f)
	if err != nil {
		fmt.Println(err)
	}
}

func getLines(source, kind string, lines []line) []line {
	res := make([]line, 0)
	for i := 0; i < len(lines); i++ {
		if lines[i].source == source && lines[i].kind == kind {
			res = append(res, lines[i])
		}
	}
	return res
}

func calculateVisibility(interference []float64) []float64 {
	lambda := 1550
	dx := 8
	winLen := lambda / dx * 1
	visibility := make([]float64, len(interference)/winLen-1)
	interferenceMax := slices.Max(interference)
	for i := 0; i < len(interference); i++ {
		interference[i] += interferenceMax
	}
	for i := 0; i < cap(visibility); i++ {
		pMax := slices.Max(interference[i*winLen : (i+1)*winLen])
		pMin := slices.Min(interference[i*winLen : (i+1)*winLen])
		visibility[i] = (pMax - pMin) / (pMax + pMin)
	}
	return visibility
}

func calculateCorrelationCoefficients(lines []line) {
	table := make(map[string][][]float64)

	for i := 0; i < len(lines); i++ {
		if !strings.Contains(lines[i].name, "time") {
			table[lines[i].source+" "+lines[i].kind] = append(table[lines[i].source+" "+lines[i].kind],
				lines[i].derivative())
		}
	}

	//for k := range table {
	//	fmt.Println(k)
	//	fmt.Println(len(table[k]))
	//}

	sources := []string{"dummy", "main"}

	for _, s := range sources {
		for i := 0; i < len(table[s+" visibility"]); i++ {
			//fmt.Println("Correlation coefficient "+s+" = ", correlationCoefficient(table[s+" power"][0], table["dummy visibility"][i]))
		}
	}
}

func prepareData(files []os.DirEntry) ([]float64, []line) {
	filesList := make([]string, 0)

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".dat") {
			filesList = append(filesList, f.Name())
		}
	}

	lines := make([]line, len(filesList))

	for i, f := range filesList {
		if strings.Contains(f, "vid") {
			lines[i] = line{yData: createYdataVid(DIR + f), name: f, kind: "visibility", index: 0}
		}
	}

	var x []float64

	for i, f := range filesList {
		if strings.Contains(f, "vid") {
			x = xFromY(lines[i].yData, LENGTH)
			break
		}
	}

	for i, f := range filesList {
		if !strings.Contains(f, "vid") {
			lines[i] = line{yData: createYdata(DIR+f, x), name: f, kind: "power", index: 1}
		}
	}

	var dx float64
	if len(x) < 2 {
		dx = 1.0
	} else {
		dx = x[1] - x[0]
	}

	for i := 0; i < len(lines); i++ {
		lines[i].dx = dx
		if strings.Contains(lines[i].name, "main") {
			lines[i].source = "main"
		} else {
			lines[i].source = "dummy"
		}
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bin") {
			filesList = append(filesList, f.Name())
			lines = append(lines, line{
				yData:  calculateVisibility(readBin(DIR + f.Name())),
				kind:   "",
				name:   f.Name(),
				source: "",
				index:  0,
				dx:     0,
			})
		}
	}
	//i1 := int(float64(len(lines[0].yData)) * 0.57)
	//i2 := int(float64(len(lines[0].yData)) * 0.575)
	//for i := 0; i < len(lines); i++ {
	//	lines[i].yData = lines[i].yData[i1:i2]
	//}

	x = xFromY(lines[0].yData, float64(len(lines[0].yData))/1_000_000)

	return x, lines
}

func readBin(s string) []float64 {
	data8 := readData(s)
	data64 := make([]float64, len(data8)/8)

	for i := 0; i < len(data64)-1; i++ {
		data64[i] = float64(int32(binary.BigEndian.Uint64(data8[i*8:(i+1)*8]))) / float64(math.MaxInt32)
	}

	return data64
}

func correlationCoefficient(x []float64, y []float64) float64 {
	xDeviation := deviation(x)
	yDeviation := deviation(y)
	numerator := sum(mul(xDeviation, yDeviation))
	denominator := math.Sqrt(sum(mul(xDeviation, xDeviation)) * sum(mul(yDeviation, yDeviation)))
	return numerator / denominator
}

func mul(x, y []float64) []float64 {
	res := make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		res[i] = x[i] * y[i]
	}
	return res
}

func deviation(x []float64) []float64 {
	res := make([]float64, len(x))
	xMean := mean(x)
	for i := 0; i < len(x); i++ {
		res[i] = x[i] - xMean
	}
	return res
}

func mean(x []float64) float64 {
	return sum(x) / float64(len(x))
}

func sum(x []float64) float64 {
	var res float64
	for i := 0; i < len(x); i++ {
		res += x[i]
	}
	return res
}

func (l line) derivative() []float64 {
	res := make([]float64, len(l.yData))
	for i := 0; i < len(l.yData)-1; i++ {
		res[i] = (l.yData[i+1] - l.yData[i]) / l.dx
	}
	res[len(l.yData)-1] = res[len(l.yData)-2]
	return res
}

func plotChart(x []float64, lines []line) *charts.Line {
	chart := charts.NewLine()

	chart.SetGlobalOptions(
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
		}), charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      0,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithLegendOpts(opts.Legend{
			Orient:       "horizontal",
			Show:         true,
			SelectedMode: "multiple",
			Type:         "scroll",
			//Right:        "-5%",
			Bottom: "-1%",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
			AxisPointer: &opts.AxisPointer{
				Type: "cross",
				Snap: true,
			},
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Top:  "0%",
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Type:  "png",
					Name:  "chart",
					Title: "Save as image",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:       true,
					YAxisIndex: "default",
					Title: map[string]string{"zoom": "area zooming",
						"back": "restore area zooming"},
				},
				DataView: &opts.ToolBoxFeatureDataView{
					Show:  true,
					Title: "Data view",
					Lang:  []string{"data view", "turn off", "refresh"},
				},
				Restore: &opts.ToolBoxFeatureRestore{
					Show:  true,
					Title: "refresh",
				},
			},
		}),
		// AXIS
		charts.WithXAxisOpts(opts.XAxis{
			SplitNumber: 20,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:  "Visibility",
			Type:  "value",
			Show:  true,
			Scale: true,
			//GridIndex: 0, // y index 0 // not required
		}),
	)

	chart.ExtendYAxis(opts.YAxis{
		Name:  "Power",
		Type:  "value",
		Show:  true,
		Scale: true,
		//GridIndex: 1, // y index 1 // not required
	})
	//chart.ExtendYAxis(opts.YAxis{
	//	Name:  "Derivative",
	//	Type:  "value",
	//	Show:  true,
	//	Scale: true,
	//})

	chart.SetXAxis(x)

	for i := 0; i < len(lines); i++ {
		chart.AddSeries(lines[i].name, generateLineItems(lines[i]),
			charts.WithLineChartOpts(opts.LineChart{YAxisIndex: int(lines[i].index)}))
	}

	return chart
}

func createYdata(s string, x []float64) []float64 {
	data8 := readData(s)
	data64 := make([]float64, len(data8)/8)

	for i := 0; i < len(data64)-1; i++ {
		data64[i] = Float64frombytes(data8[i*8 : (i+1)*8])
	}

	d := len(data64) / len(x)

	res := make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		res[i] = data64[i*d]
	}
	return res
}

func createYdataVid(s string) []float64 {
	data8 := readData(s)
	y := convertVid(data8)

	for i := 0; i < 4; i++ {
		y = compress(y)
	}

	return y
}

func compress(y []float64) []float64 {
	yy := make([]float64, len(y)/2)
	l := 5
	for i := 0; i < len(y)-l; i += 2 {
		yy[i/2] = med(y[i : i+l])
	}
	for i := len(yy) - (l / 2); i < cap(yy); i++ {
		yy[i] = yy[i-1]
	}

	return yy
}

func med(sl []float64) float64 {
	n := len(sl)
	sl2 := make([]float64, len(sl))
	copy(sl2, sl)
	sort.Float64s(sl2)
	if n%2 == 1 {
		return sl2[n/2]
	} else {
		return (sl2[n/2-1] + sl[n/2]) / 2
	}
}

func xFromY(y []float64, length float64) []float64 {
	x := make([]float64, len(y))
	for i := 0; i < len(y); i++ {
		x[i] = float64(i) / float64(len(y)) * length
	}
	return x
}

func convertVid(data8 []byte) []float64 {
	const SizeofFloat64 = 8 // bytes

	data64 := make([]float64, len(data8)/SizeofFloat64/2)

	for i := 0; i < len(data64)-1; i++ {

		// assuming little endian
		data64[i] = Float64frombytes(data8[(2*i)*SizeofFloat64 : (2*i+1)*SizeofFloat64])
	}
	return data64
}

func generateLineItems(l line) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < len(l.yData); i++ {
		items = append(items, opts.LineData{Value: l.yData[i], YAxisIndex: int(l.index)})
	}
	return items
}

func createHtml(filePath string, charts ...components.Charter) error {
	page := components.NewPage()
	page.AddCharts(charts...)

	file, createErr := os.Create(filePath)
	if createErr != nil {
		return createErr
	}
	return page.Render(io.MultiWriter(file))
}
