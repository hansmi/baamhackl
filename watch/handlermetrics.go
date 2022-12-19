package watch

import (
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

func makeHandlerTimeBuckets(cfg *config.Handler) []float64 {
	timeBucketsMin := time.Second
	timeBucketsMax := cfg.Timeout

	if timeBucketsMax <= timeBucketsMin {
		timeBucketsMax = timeBucketsMin + 1
	}

	return makeSecondsBuckets(timeBucketsMin, timeBucketsMax, 10)
}

type handlerMetricsCollector struct {
	mu sync.Mutex

	h *handler

	infoMetric prometheus.Metric

	fileChangeCount prometheus.Counter

	pendingTasksDesc *prometheus.Desc

	retryCount    prometheus.Counter
	finishedCount prometheus.Counter
	failureCount  prometheus.Counter

	commandExitCodeCount *prometheus.CounterVec
	commandWallTime      prometheus.Histogram
	commandUserTime      prometheus.Histogram
	commandSystemTime    prometheus.Histogram

	nested []prometheus.Collector
}

var _ prometheus.Collector = (*handlerMetricsCollector)(nil)

func newHandlerMetricsCollector(h *handler) *handlerMetricsCollector {
	timeBuckets := makeHandlerTimeBuckets(h.cfg)

	c := &handlerMetricsCollector{
		h: h,
	}

	c.infoMetric = prometheus.MustNewConstMetric(
		prometheus.NewDesc("handler_info",
			"Information about the handler.",
			[]string{"path"}, nil),
		prometheus.GaugeValue, 1,
		filepath.Clean(h.cfg.Path),
	)

	c.fileChangeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "file_changes_total",
		Help: "Number of reported file changes.",
	})

	c.pendingTasksDesc = prometheus.NewDesc("pending_total",
		"Number of currently waiting tasks.", nil, nil)

	c.retryCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retries_total",
		Help: "Number of retries.",
	})
	c.finishedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "finished_total",
		Help: "Total number of handled changes (including failures).",
	})
	c.failureCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "failures_total",
		Help: "Number of failures.",
	})

	c.commandExitCodeCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "command_exit_code_total",
		Help: "Number of times each command exit code occurred.",
	}, []string{"code"})
	c.commandExitCodeCount.WithLabelValues("0")

	c.commandWallTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "command_wall_time",
		Help:    "Histogram with the time taken by commands from start to end.",
		Buckets: timeBuckets,
	})
	c.commandUserTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "command_user_time",
		Help:    "Histogram with the user space time taken by commands.",
		Buckets: timeBuckets,
	})
	c.commandSystemTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "command_system_time",
		Help:    "Histogram with the system time taken by commands.",
		Buckets: timeBuckets,
	})

	c.nested = append(c.nested,
		c.fileChangeCount,

		c.commandExitCodeCount,
		c.commandWallTime,
		c.commandUserTime,
		c.commandSystemTime,

		c.retryCount,
		c.finishedCount,
		c.failureCount,
	)

	return c
}

func (c *handlerMetricsCollector) ReportFileChange() {
	c.fileChangeCount.Inc()
}

func (c *handlerMetricsCollector) ReportProcessState(exitCode int, wallTime, userTime, systemTime time.Duration) {
	c.mu.Lock()
	c.commandExitCodeCount.With(prometheus.Labels{
		"code": strconv.Itoa(exitCode),
	}).Inc()
	c.commandWallTime.Observe(wallTime.Seconds())
	c.commandUserTime.Observe(userTime.Seconds())
	c.commandSystemTime.Observe(systemTime.Seconds())
	c.mu.Unlock()
}

func (c *handlerMetricsCollector) ReportTaskRetry() {
	c.retryCount.Inc()
}

func (c *handlerMetricsCollector) ReportFinalTaskStatus(err error) {
	c.mu.Lock()
	c.finishedCount.Inc()
	if err != nil {
		c.failureCount.Inc()
	}
	c.mu.Unlock()
}

func (c *handlerMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoMetric.Desc()
	ch <- c.pendingTasksDesc

	for _, i := range c.nested {
		i.Describe(ch)
	}
}

func (c *handlerMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.infoMetric

	c.mu.Lock()
	for _, i := range c.nested {
		i.Collect(ch)
	}
	c.mu.Unlock()

	c.h.mu.Lock()
	ch <- prometheus.MustNewConstMetric(c.pendingTasksDesc, prometheus.GaugeValue, float64(len(c.h.pending)))
	c.h.mu.Unlock()
}
