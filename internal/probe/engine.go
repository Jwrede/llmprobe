package probe

import (
	"fmt"
	"time"

	"github.com/Jwrede/llmprobe/internal/config"
	"github.com/Jwrede/llmprobe/internal/provider"
	"golang.org/x/sync/errgroup"
)

type Engine struct {
	cfg       *config.Config
	providers map[string]provider.Provider
}

func NewEngine(cfg *config.Config) *Engine {
	providers := make(map[string]provider.Provider)
	for _, p := range cfg.Providers {
		key := p.DisplayName()
		switch p.Name {
		case "openai":
			providers[key] = provider.NewOpenAI(p.APIKey, p.BaseURL)
		case "anthropic":
			providers[key] = provider.NewAnthropic(p.APIKey, p.BaseURL)
		case "google":
			providers[key] = provider.NewGoogle(p.APIKey, p.BaseURL)
		case "azure":
			providers[key] = provider.NewAzure(p.APIKey, p.BaseURL, p.APIVersion)
		case "bedrock":
			providers[key] = provider.NewBedrock(p.AccessKey, p.SecretKey, p.Region, p.BaseURL)
		}
	}
	return &Engine{cfg: cfg, providers: providers}
}

type probeTask struct {
	index        int
	providerName string
	model        config.Model
}

func (e *Engine) RunAll() ([]Result, error) {
	var tasks []probeTask
	for _, p := range e.cfg.Providers {
		for _, m := range p.Models {
			tasks = append(tasks, probeTask{
				index:        len(tasks),
				providerName: p.DisplayName(),
				model:        m,
			})
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no probe targets configured")
	}

	results := make([]Result, len(tasks))

	g := new(errgroup.Group)
	g.SetLimit(e.cfg.Defaults.Concurrency)

	for _, task := range tasks {
		t := task
		g.Go(func() error {
			results[t.index] = e.runOne(t)
			return nil
		})
	}

	_ = g.Wait()

	return results, nil
}

func (e *Engine) runOne(t probeTask) Result {
	p, ok := e.providers[t.providerName]
	if !ok {
		return Result{
			Provider:  t.providerName,
			Model:     t.model.Name,
			Status:    StatusError,
			Error:     fmt.Sprintf("unknown provider %q", t.providerName),
			Timestamp: time.Now(),
		}
	}

	pc := provider.ProviderContext{
		Model:     t.model.Name,
		Prompt:    t.model.Prompt,
		MaxTokens: t.model.MaxTokens,
		Timeout:   e.cfg.Defaults.Timeout.Duration,
	}

	pr, err := p.Probe(pc)

	r := Result{
		Provider:  t.providerName,
		Model:     t.model.Name,
		Timestamp: time.Now(),
	}

	if err != nil {
		r.Status = StatusError
		r.Error = err.Error()
		return r
	}

	r.TTFT = pr.TTFT
	r.TotalLatency = pr.TotalLatency
	r.TokenCount = pr.TokenCount

	if pr.TokenCount == 0 {
		r.Status = StatusDegraded
		r.Error = "received HTTP 200 but no content tokens"
		return r
	}

	genTime := pr.TotalLatency - pr.TTFT
	if genTime > time.Millisecond {
		r.TokensPerSec = float64(pr.TokenCount) / genTime.Seconds()
	}

	r.Status = e.applyThresholds(r, t.model.Thresholds)

	return r
}

func (e *Engine) applyThresholds(r Result, th config.Thresholds) Status {
	if th.MaxTTFT.Duration > 0 && r.TTFT > th.MaxTTFT.Duration {
		return StatusDegraded
	}
	if th.MaxLatency.Duration > 0 && r.TotalLatency > th.MaxLatency.Duration {
		return StatusDegraded
	}
	if th.MinTokensPerS > 0 && r.TokensPerSec < th.MinTokensPerS {
		return StatusDegraded
	}
	return StatusHealthy
}
