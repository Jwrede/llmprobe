package output

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Jwrede/llmprobe/internal/probe"
)

func PrintTable(results []probe.Result) {
	WriteTable(os.Stdout, results)
}

func WriteTable(w io.Writer, results []probe.Result) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "Provider\tModel\tStatus\tTTFT\tLatency\tTok/s\tTokens\tError")
	fmt.Fprintln(tw, "--------\t-----\t------\t----\t-------\t-----\t------\t-----")

	for _, r := range results {
		ttft := ""
		latency := ""
		tps := ""
		tokens := ""
		errMsg := ""

		if r.Status != probe.StatusError {
			ttft = fmtDuration(r.TTFT)
			latency = fmtDuration(r.TotalLatency)
			tps = fmt.Sprintf("%.1f", r.TokensPerSec)
			tokens = fmt.Sprintf("%d", r.TokenCount)
		}
		if r.Error != "" {
			errMsg = r.Error
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Provider, r.Model, r.Status,
			ttft, latency, tps, tokens, errMsg)
	}

	tw.Flush()
}

func fmtDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func PrintSummary(results []probe.Result) {
	healthy := 0
	degraded := 0
	errors := 0
	for _, r := range results {
		switch r.Status {
		case probe.StatusHealthy:
			healthy++
		case probe.StatusDegraded:
			degraded++
		case probe.StatusError:
			errors++
		}
	}
	fmt.Printf("\n%d healthy, %d degraded, %d errors\n", healthy, degraded, errors)
}
