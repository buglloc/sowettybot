package renderer

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"

	"github.com/buglloc/sowettybot/internal/models"
)

type HistoryRenderer struct {
}

func NewHistoryRenderer() *HistoryRenderer {
	return &HistoryRenderer{}
}

func (h *HistoryRenderer) Rates(rates models.Rates) (string, error) {
	var out strings.Builder
	if err := renderTemplate(&out, "rates.gotmpl", rates); err != nil {
		return "", fmt.Errorf("render failed: %w", err)
	}

	return out.String(), nil
}

func (h *HistoryRenderer) Log(entries []models.History) (string, error) {
	var out strings.Builder
	if err := renderTemplate(&out, "log.gotmpl", entries); err != nil {
		return "", fmt.Errorf("render failed: %w", err)
	}

	return out.String(), nil
}

func (h *HistoryRenderer) Graph(entries []models.History, out io.Writer, cfg *GraphConfig) (startDate time.Time, endDate time.Time, err error) {
	var series []chart.TimeSeries
	var prevData models.History
	for _, entry := range entries {
		if len(series) == 0 {
			series = make([]chart.TimeSeries, len(entry.Names))
			for i, name := range entry.Names {
				series[i].Name = name
				color := chart.GetDefaultColor(i)
				series[i].Style = chart.Style{
					Show:        true,
					StrokeColor: color,
					FillColor:   color.WithAlpha(50),
				}
			}
		}

		// fix up zeroes
		ok := true
		for i, val := range entry.Values {
			if val == 0.0 {
				if prevData.When.IsZero() {
					ok = false
				} else {
					entry.Values[i] = prevData.Values[i]
				}
			}
		}
		if !ok {
			continue
		}

		for i := range entry.Values {
			series[i].XValues = append(series[i].XValues, entry.When)
			series[i].YValues = append(series[i].YValues, entry.Values[i])
		}

		if startDate.IsZero() {
			startDate = entry.When
		}
		endDate = entry.When
		prevData = entry
	}

	seriesesSize := 2
	if cfg.showSMA {
		seriesesSize++
	}

	graph := chart.Chart{
		Width:  cfg.width,
		Height: cfg.height,
		XAxis: chart.XAxis{
			Name:           "date",
			Style:          chart.StyleShow(),
			ValueFormatter: chart.TimeValueFormatterWithFormat("Mon 15:04"),
			TickPosition:   chart.TickPositionUnderTick,
		},
		YAxis: chart.YAxis{
			Style:    chart.StyleShow(),
			AxisType: chart.YAxisSecondary,
		},
		Series: make([]chart.Series, seriesesSize*len(series)),
	}
	for i := 0; i < len(series); i++ {
		graph.Series[i*seriesesSize] = series[i]

		if cfg.showSMA {
			graph.Series[i*seriesesSize+1] = &chart.SMASeries{
				Name:   fmt.Sprintf("%s (sma)", series[i].Name),
				Period: len(entries) / 5,
				Style: chart.Style{
					Show:            true,
					StrokeColor:     series[i].Style.StrokeColor,
					StrokeWidth:     3.0,
					StrokeDashArray: []float64{5.0, 5.0},
				},
				InnerSeries: series[i],
			}
		}

		graph.Series[i*seriesesSize+seriesesSize-1] = chart.LastValueAnnotation(series[i])
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph, chart.Style{
			FillColor:   drawing.ColorWhite,
			FontColor:   chart.DefaultTextColor,
			FontSize:    8.0,
			StrokeColor: chart.ColorTransparent,
		}),
	}

	return startDate, endDate, graph.Render(chart.PNG, out)
}
