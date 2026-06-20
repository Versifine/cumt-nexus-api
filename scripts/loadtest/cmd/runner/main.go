package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type runnerConfig struct {
	BaseURL        string        `json:"base_url"`
	Duration       time.Duration `json:"duration"`
	Warmup         time.Duration `json:"warmup"`
	VUs            int           `json:"vus"`
	Users          int           `json:"users"`
	Communities    int           `json:"communities"`
	Posts          int           `json:"posts"`
	Comments       int           `json:"comments"`
	Notifications  int           `json:"notifications"`
	Reports        int           `json:"reports"`
	Include        string        `json:"include"`
	Exclude        string        `json:"exclude"`
	TargetRPS      float64       `json:"target_rps"`
	ReportJSON     string        `json:"report_json"`
	ReportMarkdown string        `json:"report_markdown"`
}

type requestSpec struct {
	Name   string
	Method string
	Path   func(r *rand.Rand, cfg runnerConfig) string
	Body   func(r *rand.Rand, cfg runnerConfig) []byte
	Auth   authMode
	Weight int
}

type authMode int

const (
	authNone authMode = iota
	authPrimary
	authRandomUser
)

type result struct {
	Endpoint string
	Status   int
	Duration time.Duration
	Error    string
	Bytes    int64
}

type report struct {
	StartedAt       time.Time                 `json:"started_at"`
	FinishedAt      time.Time                 `json:"finished_at"`
	ElapsedSeconds  float64                   `json:"elapsed_seconds"`
	BaseURL         string                    `json:"base_url"`
	VUs             int                       `json:"vus"`
	WarmupSeconds   float64                   `json:"warmup_seconds"`
	DurationSeconds float64                   `json:"duration_seconds"`
	TargetRPS       float64                   `json:"target_rps,omitempty"`
	Included        []string                  `json:"included"`
	Excluded        []string                  `json:"excluded"`
	Environment     environmentInfo           `json:"environment"`
	Dataset         datasetInfo               `json:"dataset"`
	Overall         metricSummary             `json:"overall"`
	Endpoints       map[string]metricSummary  `json:"endpoints"`
	StatusCodes     map[string]map[int]int    `json:"status_codes"`
	Errors          map[string]map[string]int `json:"errors"`
}

type environmentInfo struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Go     string `json:"go"`
	NumCPU int    `json:"num_cpu"`
	Tool   string `json:"tool"`
}

type datasetInfo struct {
	Users       int `json:"users"`
	Communities int `json:"communities"`
	Posts       int `json:"posts"`
	Comments    int `json:"comments"`
}

type metricSummary struct {
	Requests  int     `json:"requests"`
	Failures  int     `json:"failures"`
	ErrorRate float64 `json:"error_rate"`
	RPS       float64 `json:"rps"`
	Bytes     int64   `json:"bytes"`
	MinMS     float64 `json:"min_ms"`
	P50MS     float64 `json:"p50_ms"`
	P95MS     float64 `json:"p95_ms"`
	P99MS     float64 `json:"p99_ms"`
	MaxMS     float64 `json:"max_ms"`
	AvgMS     float64 `json:"avg_ms"`
}

func main() {
	cfg := runnerConfig{}
	flag.StringVar(&cfg.BaseURL, "base-url", "http://127.0.0.1:18080", "API base URL")
	flag.DurationVar(&cfg.Duration, "duration", 1*time.Minute, "measured load-test duration")
	flag.DurationVar(&cfg.Warmup, "warmup", 5*time.Second, "warmup duration; results are discarded")
	flag.IntVar(&cfg.VUs, "vus", 50, "concurrent virtual users")
	flag.IntVar(&cfg.Users, "users", 1000, "seeded user count")
	flag.IntVar(&cfg.Communities, "communities", 50, "seeded community count")
	flag.IntVar(&cfg.Posts, "posts", 20000, "seeded post count")
	flag.IntVar(&cfg.Comments, "comments", 80000, "seeded comment count")
	flag.IntVar(&cfg.Notifications, "notifications", 12000, "seeded notification count")
	flag.IntVar(&cfg.Reports, "reports", 3000, "seeded report count")
	flag.StringVar(&cfg.Include, "include", "", "comma-separated endpoint names to include; empty includes the full request mix")
	flag.StringVar(&cfg.Exclude, "exclude", "", "comma-separated endpoint names to exclude from the request mix")
	flag.Float64Var(&cfg.TargetRPS, "target-rps", 0, "fixed request issue rate; 0 runs max-throughput VU loops")
	flag.StringVar(&cfg.ReportJSON, "out-json", "", "write JSON report to this path")
	flag.StringVar(&cfg.ReportMarkdown, "out-md", "", "write Markdown report to this path")
	flag.Parse()

	if err := validateRunnerConfig(cfg); err != nil {
		fatal(err)
	}

	appCfg, err := config.Load()
	if err != nil {
		fatal(fmt.Errorf("load config: %w", err))
	}

	tokens, err := issueTokens(appCfg, cfg.Users)
	if err != nil {
		fatal(err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.VUs * 4,
			MaxIdleConnsPerHost: cfg.VUs * 4,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	if cfg.Warmup > 0 {
		if _, err := runFor(context.Background(), client, cfg, tokens, cfg.Warmup, false); err != nil {
			fatal(err)
		}
	}

	startedAt := time.Now().UTC()
	results, err := runFor(context.Background(), client, cfg, tokens, cfg.Duration, true)
	if err != nil {
		fatal(err)
	}
	finishedAt := time.Now().UTC()

	rep := buildReport(cfg, startedAt, finishedAt, results)
	if err := writeReports(rep, cfg); err != nil {
		fatal(err)
	}
	printConsoleSummary(rep)
}

func validateRunnerConfig(cfg runnerConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("base-url is required")
	}
	if cfg.Duration <= 0 || cfg.Warmup < 0 {
		return fmt.Errorf("duration must be positive and warmup cannot be negative")
	}
	if cfg.VUs <= 0 {
		return fmt.Errorf("vus must be positive")
	}
	if cfg.Users < 20 || cfg.Communities < 1 || cfg.Posts < 1 || cfg.Comments < 1 {
		return fmt.Errorf("dataset counts are invalid")
	}
	if cfg.Notifications < 0 || cfg.Reports < 0 {
		return fmt.Errorf("notification and report counts cannot be negative")
	}
	if cfg.TargetRPS < 0 {
		return fmt.Errorf("target-rps cannot be negative")
	}
	if len(requestSpecs(cfg)) == 0 {
		return fmt.Errorf("include/exclude filters leave no endpoints to run")
	}
	return nil
}

func issueTokens(cfg *config.Config, users int) ([]string, error) {
	issuer := authtoken.NewJWTIssuer(cfg.Auth.TokenSecret, cfg.App.Name, cfg.Auth.AccessTokenTTL)
	count := users
	if count > 200 {
		count = 200
	}
	tokens := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		userID, err := userdomain.NewUserID(loadtestUUID("1000", i))
		if err != nil {
			return nil, err
		}
		token, _, _, err := issuer.IssueAccessToken(userID, time.Now().UTC().Add(-time.Minute))
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func runFor(parent context.Context, client *http.Client, cfg runnerConfig, tokens []string, duration time.Duration, collect bool) ([]result, error) {
	if cfg.TargetRPS > 0 {
		return runFixedRateFor(parent, client, cfg, tokens, duration, collect)
	}
	return runMaxThroughputFor(parent, client, cfg, tokens, duration, collect)
}

func runMaxThroughputFor(parent context.Context, client *http.Client, cfg runnerConfig, tokens []string, duration time.Duration, collect bool) ([]result, error) {
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()

	resultCh := make(chan result, cfg.resultBufferSize(duration))
	var wg sync.WaitGroup
	for i := 0; i < cfg.VUs; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)*7919))
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				res := executeOne(ctx, client, cfg, tokens, r)
				if collect {
					select {
					case resultCh <- res:
					case <-ctx.Done():
						return
					}
				}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]result, 0)
	for res := range resultCh {
		results = append(results, res)
	}
	return results, nil
}

func runFixedRateFor(parent context.Context, client *http.Client, cfg runnerConfig, tokens []string, duration time.Duration, collect bool) ([]result, error) {
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()

	interval := time.Duration(float64(time.Second) / cfg.TargetRPS)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	resultCh := make(chan result, cfg.resultBufferSize(duration))
	semaphore := make(chan struct{}, cfg.VUs)
	var wg sync.WaitGroup
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	emit := func(res result) {
		if !collect {
			return
		}
		select {
		case resultCh <- res:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(resultCh)
			results := make([]result, 0)
			for res := range resultCh {
				results = append(results, res)
			}
			return results, nil
		case <-ticker.C:
			spec := pickSpec(r, requestSpecs(cfg))
			select {
			case semaphore <- struct{}{}:
				seed := r.Int63()
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-semaphore }()
					res := executeSpec(ctx, client, cfg, tokens, spec, rand.New(rand.NewSource(seed)))
					emit(res)
				}()
			default:
				emit(result{Endpoint: spec.Name, Error: "client_backpressure"})
			}
		}
	}
}

func executeOne(ctx context.Context, client *http.Client, cfg runnerConfig, tokens []string, r *rand.Rand) result {
	return executeSpec(ctx, client, cfg, tokens, pickSpec(r, requestSpecs(cfg)), r)
}

func (cfg runnerConfig) resultBufferSize(duration time.Duration) int {
	size := cfg.VUs * 16
	if cfg.TargetRPS > 0 {
		expected := int(cfg.TargetRPS*duration.Seconds()) + cfg.VUs*16
		if expected > size {
			size = expected
		}
	}
	if size < 1 {
		return 1
	}
	return size
}

func executeSpec(ctx context.Context, client *http.Client, cfg runnerConfig, tokens []string, spec requestSpec, r *rand.Rand) result {
	var body []byte
	if spec.Body != nil {
		body = spec.Body(r, cfg)
	}
	requestCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, spec.Method, strings.TrimRight(cfg.BaseURL, "/")+spec.Path(r, cfg), bytes.NewReader(body))
	if err != nil {
		return result{Endpoint: spec.Name, Error: err.Error()}
	}
	req.Header.Set("User-Agent", "cumt-nexus-loadtest/1.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	switch spec.Auth {
	case authPrimary:
		req.Header.Set("Authorization", "Bearer "+tokens[0])
	case authRandomUser:
		req.Header.Set("Authorization", "Bearer "+tokens[r.Intn(len(tokens))])
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return result{Endpoint: spec.Name, Duration: elapsed, Error: err.Error()}
	}
	defer resp.Body.Close()
	n, _ := io.Copy(io.Discard, resp.Body)
	res := result{Endpoint: spec.Name, Status: resp.StatusCode, Duration: elapsed, Bytes: n}
	if resp.StatusCode >= 400 {
		res.Error = fmt.Sprintf("http_%d", resp.StatusCode)
	}
	return res
}

func pickSpec(r *rand.Rand, specs []requestSpec) requestSpec {
	total := 0
	for _, spec := range specs {
		total += spec.Weight
	}
	n := r.Intn(total)
	for _, spec := range specs {
		if n < spec.Weight {
			return spec
		}
		n -= spec.Weight
	}
	return specs[len(specs)-1]
}

func requestSpecs(cfg runnerConfig) []requestSpec {
	specs := []requestSpec{
		{
			Name:   "posts_feed_new",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				return fmt.Sprintf("/api/v1/posts?source=all&sort=new&limit=20&offset=%d", randomOffset(r, cfg.Posts, 20))
			},
			Auth:   authNone,
			Weight: 18,
		},
		{
			Name:   "posts_feed_hot",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				return fmt.Sprintf("/api/v1/posts?source=recommended&sort=hot&t=day&limit=20&offset=%d", randomOffset(r, cfg.Posts, 20))
			},
			Auth:   authRandomUser,
			Weight: 10,
		},
		{
			Name:   "community_posts",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				community := r.Intn(cfg.Communities) + 1
				return fmt.Sprintf("/api/v1/communities/lt-community-%04d/posts?sort=new&limit=20&offset=%d", community, randomOffset(r, cfg.Posts/cfg.Communities, 20))
			},
			Auth:   authRandomUser,
			Weight: 14,
		},
		{
			Name:   "post_detail",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				return "/api/v1/posts/" + loadtestUUID("3000", r.Intn(cfg.Posts)+1)
			},
			Auth:   authRandomUser,
			Weight: 15,
		},
		{
			Name:   "comment_tree",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				postID := r.Intn(cfg.Posts) + 1
				return fmt.Sprintf("/api/v1/posts/%s/comments?view=tree&sort=top&limit=20&offset=0&max_depth=6", loadtestUUID("3000", postID))
			},
			Auth:   authRandomUser,
			Weight: 14,
		},
		{
			Name:   "search_all",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				queries := []string{"nexus", "moderation", "feed", "community", "notification"}
				return fmt.Sprintf("/api/v1/search?q=%s&scope=all&limit=10&offset=0", queries[r.Intn(len(queries))])
			},
			Auth:   authNone,
			Weight: 10,
		},
		{
			Name:   "notifications_interactions",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				return fmt.Sprintf("/api/v1/notifications?category=interactions&status=all&limit=20&offset=%d", randomOffset(r, cfg.Notifications, 20))
			},
			Auth:   authPrimary,
			Weight: 8,
		},
		{
			Name:   "admin_mod_queue",
			Method: http.MethodGet,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				return fmt.Sprintf("/api/v1/admin/mod-queues?queue=reports&limit=20&offset=%d", randomOffset(r, cfg.Reports, 20))
			},
			Auth:   authPrimary,
			Weight: 6,
		},
		{
			Name:   "post_vote_write",
			Method: http.MethodPut,
			Path: func(r *rand.Rand, cfg runnerConfig) string {
				return "/api/v1/posts/" + loadtestUUID("3000", r.Intn(cfg.Posts)+1) + "/vote"
			},
			Body: func(r *rand.Rand, cfg runnerConfig) []byte {
				if r.Intn(8) == 0 {
					return []byte(`{"value":-1}`)
				}
				return []byte(`{"value":1}`)
			},
			Auth:   authRandomUser,
			Weight: 5,
		},
	}
	included := endpointSet(cfg.Include)
	excluded := endpointSet(cfg.Exclude)
	filtered := make([]requestSpec, 0, len(specs))
	for _, spec := range specs {
		if len(included) > 0 && !included[spec.Name] {
			continue
		}
		if excluded[spec.Name] {
			continue
		}
		filtered = append(filtered, spec)
	}
	if len(included) == 0 && len(excluded) == 0 {
		return specs
	}
	return filtered
}

func endpointSet(raw string) map[string]bool {
	values := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			values[name] = true
		}
	}
	return values
}

func randomOffset(r *rand.Rand, total int, limit int) int {
	if total <= limit {
		return 0
	}
	maxPage := total / limit
	if maxPage > 200 {
		maxPage = 200
	}
	return r.Intn(maxPage) * limit
}

func loadtestUUID(group string, i int) string {
	return fmt.Sprintf("00000000-0000-%s-0000-%012d", group, i)
}

func buildReport(cfg runnerConfig, startedAt time.Time, finishedAt time.Time, results []result) report {
	endpointResults := map[string][]result{}
	statusCodes := map[string]map[int]int{}
	errorsByEndpoint := map[string]map[string]int{}
	for _, res := range results {
		endpointResults[res.Endpoint] = append(endpointResults[res.Endpoint], res)
		if _, ok := statusCodes[res.Endpoint]; !ok {
			statusCodes[res.Endpoint] = map[int]int{}
		}
		statusCodes[res.Endpoint][res.Status]++
		if res.Error != "" {
			if _, ok := errorsByEndpoint[res.Endpoint]; !ok {
				errorsByEndpoint[res.Endpoint] = map[string]int{}
			}
			errorsByEndpoint[res.Endpoint][res.Error]++
		}
	}

	endpoints := map[string]metricSummary{}
	for endpoint, values := range endpointResults {
		endpoints[endpoint] = summarize(values, cfg.Duration)
	}
	return report{
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		ElapsedSeconds:  finishedAt.Sub(startedAt).Seconds(),
		BaseURL:         cfg.BaseURL,
		VUs:             cfg.VUs,
		WarmupSeconds:   cfg.Warmup.Seconds(),
		DurationSeconds: cfg.Duration.Seconds(),
		TargetRPS:       cfg.TargetRPS,
		Included:        sortedEndpointSet(cfg.Include),
		Excluded:        sortedExcluded(cfg.Exclude),
		Environment: environmentInfo{
			OS:     runtime.GOOS,
			Arch:   runtime.GOARCH,
			Go:     runtime.Version(),
			NumCPU: runtime.NumCPU(),
			Tool:   "scripts/loadtest/cmd/runner",
		},
		Dataset: datasetInfo{
			Users:       cfg.Users,
			Communities: cfg.Communities,
			Posts:       cfg.Posts,
			Comments:    cfg.Comments,
		},
		Overall:     summarize(results, cfg.Duration),
		Endpoints:   endpoints,
		StatusCodes: statusCodes,
		Errors:      errorsByEndpoint,
	}
}

func sortedExcluded(raw string) []string {
	return sortedEndpointSet(raw)
}

func sortedEndpointSet(raw string) []string {
	values := make([]string, 0)
	for name := range endpointSet(raw) {
		values = append(values, name)
	}
	sort.Strings(values)
	return values
}

func summarize(results []result, duration time.Duration) metricSummary {
	if len(results) == 0 {
		return metricSummary{}
	}
	durations := make([]float64, 0, len(results))
	var failures int
	var bytesRead int64
	var sum float64
	for _, res := range results {
		ms := float64(res.Duration.Microseconds()) / 1000
		durations = append(durations, ms)
		sum += ms
		bytesRead += res.Bytes
		if res.Error != "" {
			failures++
		}
	}
	sort.Float64s(durations)
	requests := len(results)
	return metricSummary{
		Requests:  requests,
		Failures:  failures,
		ErrorRate: float64(failures) / float64(requests),
		RPS:       float64(requests) / duration.Seconds(),
		Bytes:     bytesRead,
		MinMS:     durations[0],
		P50MS:     percentile(durations, 0.50),
		P95MS:     percentile(durations, 0.95),
		P99MS:     percentile(durations, 0.99),
		MaxMS:     durations[len(durations)-1],
		AvgMS:     sum / float64(requests),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := p * float64(len(sorted)-1)
	lower := int(pos)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := pos - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func writeReports(rep report, cfg runnerConfig) error {
	if cfg.ReportJSON != "" {
		if err := os.MkdirAll(parentDir(cfg.ReportJSON), 0755); err != nil {
			return err
		}
		f, err := os.Create(cfg.ReportJSON)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(rep); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	if cfg.ReportMarkdown != "" {
		if err := os.MkdirAll(parentDir(cfg.ReportMarkdown), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(cfg.ReportMarkdown, []byte(markdownReport(rep)), 0644); err != nil {
			return err
		}
	}
	return nil
}

func markdownReport(rep report) string {
	var b strings.Builder
	b.WriteString("# CUMT Nexus API Load Test Report\n\n")
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "- Time: `%s` to `%s`\n", rep.StartedAt.Format(time.RFC3339), rep.FinishedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- Base URL: `%s`\n", rep.BaseURL)
	fmt.Fprintf(&b, "- Load: `%d` VUs, `%.0fs` warmup, `%.0fs` measured duration\n", rep.VUs, rep.WarmupSeconds, rep.DurationSeconds)
	if rep.TargetRPS > 0 {
		fmt.Fprintf(&b, "- Target issue rate: `%.2f req/s`\n", rep.TargetRPS)
	}
	fmt.Fprintf(&b, "- Dataset: `%d` users, `%d` communities, `%d` posts, `%d` comments\n", rep.Dataset.Users, rep.Dataset.Communities, rep.Dataset.Posts, rep.Dataset.Comments)
	fmt.Fprintf(&b, "- Environment: `%s/%s`, `%s`, `%d` logical CPUs\n\n", rep.Environment.OS, rep.Environment.Arch, rep.Environment.Go, rep.Environment.NumCPU)
	if len(rep.Included) > 0 {
		fmt.Fprintf(&b, "- Included endpoints: `%s`\n\n", strings.Join(rep.Included, "`, `"))
	}
	if len(rep.Excluded) > 0 {
		fmt.Fprintf(&b, "- Excluded endpoints: `%s`\n\n", strings.Join(rep.Excluded, "`, `"))
	}

	b.WriteString("## Overall Result\n\n")
	b.WriteString("| Requests | RPS | Error Rate | p50 | p95 | p99 | Max |\n")
	b.WriteString("|---:|---:|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| %d | %.2f | %.2f%% | %.2f ms | %.2f ms | %.2f ms | %.2f ms |\n\n",
		rep.Overall.Requests,
		rep.Overall.RPS,
		rep.Overall.ErrorRate*100,
		rep.Overall.P50MS,
		rep.Overall.P95MS,
		rep.Overall.P99MS,
		rep.Overall.MaxMS,
	)

	b.WriteString("## Endpoint Results\n\n")
	b.WriteString("| Endpoint | Requests | RPS | Errors | p50 | p95 | p99 | Max |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
	names := make([]string, 0, len(rep.Endpoints))
	for name := range rep.Endpoints {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m := rep.Endpoints[name]
		fmt.Fprintf(&b, "| `%s` | %d | %.2f | %.2f%% | %.2f ms | %.2f ms | %.2f ms | %.2f ms |\n",
			name,
			m.Requests,
			m.RPS,
			m.ErrorRate*100,
			m.P50MS,
			m.P95MS,
			m.P99MS,
			m.MaxMS,
		)
	}
	b.WriteString("\n## Status Codes\n\n")
	for _, name := range names {
		b.WriteString("- `" + name + "`: ")
		b.WriteString(formatStatusCodes(rep.StatusCodes[name]))
		b.WriteByte('\n')
	}
	b.WriteString("\n## Notes\n\n")
	b.WriteString("- This is a local reproducible benchmark, not production traffic evidence.\n")
	b.WriteString("- The runner treats any HTTP status code >= 400 as an error.\n")
	b.WriteString("- The request mix includes public reads, viewer-aware reads, notification reads, admin mod queue reads, and post vote writes.\n")
	b.WriteString("- Use the JSON artifact for exact raw aggregate values if the Markdown is copied into a resume note.\n")
	return b.String()
}

func formatStatusCodes(values map[int]int) string {
	if len(values) == 0 {
		return "`none`"
	}
	codes := make([]int, 0, len(values))
	for code := range values {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		parts = append(parts, fmt.Sprintf("`%d=%d`", code, values[code]))
	}
	return strings.Join(parts, ", ")
}

func parentDir(path string) string {
	idx := strings.LastIndexAny(path, `/\`)
	if idx < 0 {
		return "."
	}
	return path[:idx]
}

func printConsoleSummary(rep report) {
	fmt.Printf("requests=%d rps=%.2f error_rate=%.2f%% p95=%.2fms p99=%.2fms\n",
		rep.Overall.Requests,
		rep.Overall.RPS,
		rep.Overall.ErrorRate*100,
		rep.Overall.P95MS,
		rep.Overall.P99MS,
	)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "loadtest runner:", err)
	os.Exit(1)
}
