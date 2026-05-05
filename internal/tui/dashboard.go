package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/mum4k/termdash/widgets/text"

	"github.com/Jwrede/llmprobe/internal/probe"
)

var modelColors = []cell.Color{
	cell.ColorNumber(33),
	cell.ColorNumber(208),
	cell.ColorNumber(40),
	cell.ColorNumber(135),
	cell.ColorNumber(196),
	cell.ColorNumber(51),
	cell.ColorNumber(220),
	cell.ColorNumber(99),
}

type modelHistory struct {
	ttft    []float64
	latency []float64
	tps     []float64
	tokens  []int
	errors  int
	last    *probe.Result
}

type Dashboard struct {
	mu      sync.Mutex
	history map[string]*modelHistory
	order   []string
	chart   *linechart.LineChart
	stats   *text.Text
	title   *text.Text
}

func New() (*Dashboard, error) {
	chart, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorGray)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGray)),
	)
	if err != nil {
		return nil, err
	}

	stats, err := text.New(text.RollContent())
	if err != nil {
		return nil, err
	}

	title, err := text.New()
	if err != nil {
		return nil, err
	}

	return &Dashboard{
		history: make(map[string]*modelHistory),
		chart:   chart,
		stats:   stats,
		title:   title,
	}, nil
}

type jsonRecord struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Status       string  `json:"status"`
	TTFTMs       int     `json:"ttft_ms"`
	LatencyMs    int     `json:"latency_ms"`
	TokensPerSec float64 `json:"tokens_per_sec"`
	TokenCount   int     `json:"token_count"`
}

func (d *Dashboard) LoadJSONL(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	d.mu.Lock()
	defer d.mu.Unlock()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	loaded := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var rec jsonRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}

		key := rec.Provider + "/" + rec.Model
		h, ok := d.history[key]
		if !ok {
			h = &modelHistory{}
			d.history[key] = h
			d.order = append(d.order, key)
		}

		if rec.Status == "error" {
			h.errors++
			continue
		}

		h.ttft = append(h.ttft, float64(rec.TTFTMs))
		h.latency = append(h.latency, float64(rec.LatencyMs))
		h.tps = append(h.tps, rec.TokensPerSec)
		h.tokens = append(h.tokens, rec.TokenCount)
		loaded++
	}

	sort.Strings(d.order)
	d.redraw()
	return scanner.Err()
}

func (d *Dashboard) Update(results []probe.Result) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, r := range results {
		key := r.Provider + "/" + r.Model
		h, ok := d.history[key]
		if !ok {
			h = &modelHistory{}
			d.history[key] = h
			d.order = append(d.order, key)
			sort.Strings(d.order)
		}

		if r.Status == probe.StatusError {
			h.errors++
			h.last = &r
			continue
		}

		h.ttft = append(h.ttft, float64(r.TTFT.Milliseconds()))
		h.latency = append(h.latency, float64(r.TotalLatency.Milliseconds()))
		h.tps = append(h.tps, r.TokensPerSec)
		h.tokens = append(h.tokens, r.TokenCount)
		h.last = &r
	}

	d.redraw()
}

const maxChartPoints = 200

func downsample(vals []float64, maxPoints int) []float64 {
	if len(vals) <= maxPoints {
		return vals
	}
	bucketSize := len(vals) / maxPoints
	out := make([]float64, 0, maxPoints)
	for i := 0; i < len(vals); i += bucketSize {
		end := i + bucketSize
		if end > len(vals) {
			end = len(vals)
		}
		bucket := vals[i:end]
		sum := 0.0
		for _, v := range bucket {
			sum += v
		}
		out = append(out, sum/float64(len(bucket)))
	}
	return out
}

func (d *Dashboard) redraw() {
	for i, key := range d.order {
		h := d.history[key]
		if len(h.ttft) == 0 {
			continue
		}
		color := modelColors[i%len(modelColors)]
		d.chart.Series(key, downsample(h.ttft, maxChartPoints),
			linechart.SeriesCellOpts(cell.FgColor(color)),
		)
	}

	d.stats.Reset()
	d.stats.Write(fmt.Sprintf("  %-38s %8s %8s %8s %8s %5s %5s\n",
		"Model", "TTFT", "p95", "Lat", "Tok/s", "Err", "N"),
		text.WriteCellOpts(cell.FgColor(cell.ColorWhite), cell.Bold()),
	)
	d.stats.Write(fmt.Sprintf("  %s\n", "--------------------------------------------------------------------------------"),
		text.WriteCellOpts(cell.FgColor(cell.ColorGray)),
	)

	for i, key := range d.order {
		h := d.history[key]
		color := modelColors[i%len(modelColors)]
		n := len(h.ttft)

		if n == 0 {
			d.stats.Write(fmt.Sprintf("  %-38s %8s %8s %8s %8s %5d %5d\n",
				key, "-", "-", "-", "-", h.errors, 0),
				text.WriteCellOpts(cell.FgColor(color)),
			)
			continue
		}

		p50 := median(h.ttft)
		p95 := percentile(h.ttft, 0.95)
		latMed := median(h.latency)
		tpsMed := median(h.tps)

		d.stats.Write(fmt.Sprintf("  %-38s %7.0fms %7.0fms %7.0fms %7.1f %5d %5d\n",
			key, p50, p95, latMed, tpsMed, h.errors, n),
			text.WriteCellOpts(cell.FgColor(color)),
		)
	}

	total := 0
	healthy := 0
	degraded := 0
	errors := 0
	for _, h := range d.history {
		n := len(h.ttft)
		total += n + h.errors
		healthy += n
		errors += h.errors
		if h.last != nil && h.last.Status == probe.StatusDegraded {
			degraded++
		}
	}

	d.title.Reset()
	d.title.Write(fmt.Sprintf("  %d probes | %d healthy | %d degraded | %d errors | press q to quit",
		total, healthy, degraded, errors),
		text.WriteCellOpts(cell.FgColor(cell.ColorGray)),
	)
}

func (d *Dashboard) Run(ctx context.Context, cancel context.CancelFunc) error {
	t, err := tcell.New()
	if err != nil {
		return fmt.Errorf("terminal init: %w", err)
	}
	defer t.Close()

	c, err := container.New(
		t,
		container.Border(linestyle.Round),
		container.BorderTitle(" llmprobe "),
		container.BorderTitleAlignCenter(),
		container.BorderColor(cell.ColorGray),
		container.SplitHorizontal(
			container.Top(
				container.SplitHorizontal(
					container.Top(
						container.Border(linestyle.Light),
						container.BorderTitle(" TTFT (ms) over time "),
						container.BorderColor(cell.ColorGray),
						container.PlaceWidget(d.chart),
					),
					container.Bottom(
						container.PlaceWidget(d.title),
					),
					container.SplitPercent(95),
				),
			),
			container.Bottom(
				container.Border(linestyle.Light),
				container.BorderTitle(" Statistics "),
				container.BorderColor(cell.ColorGray),
				container.PlaceWidget(d.stats),
			),
			container.SplitPercent(60),
		),
	)
	if err != nil {
		return fmt.Errorf("container: %w", err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' || k.Key == keyboard.KeyCtrlC || k.Key == keyboard.KeyEsc {
			cancel()
		}
	}

	return termdash.Run(ctx, t, c,
		termdash.KeyboardSubscriber(quitter),
		termdash.RedrawInterval(500*time.Millisecond),
	)
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
