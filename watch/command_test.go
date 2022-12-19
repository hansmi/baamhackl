package watch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap/zaptest"
)

func TestServeMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "test_value",
	}, func() float64 { return 1 }))

	u, stop, err := listenAndServeMetrics(logger, "127.0.0.1:0", reg)
	if err != nil {
		t.Fatalf("listenAndServeMetrics() failed: %v", err)
	}

	if err := testutil.ScrapeAndCompare(u, strings.NewReader(`
		# TYPE test_value gauge
		test_value 1
		`),
		"test_value",
	); err != nil {
		t.Errorf("ScrapeAndCompare failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := stop(ctx); err != nil {
		t.Errorf("stop() failed: %v", err)
	}
}
