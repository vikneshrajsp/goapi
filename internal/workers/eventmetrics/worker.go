package eventmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/kafka-go"
	"goapi/internal/messaging/events"
)

var (
	coinBalanceEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goapi_coinbalance_events_total",
		Help: "Total number of coin balance changed events consumed by metrics worker.",
	}, []string{"username"})
	coinBalanceDelta = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "goapi_coinbalance_delta",
		Help:    "Absolute delta in coin balance for consumed events.",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
	})
	coinBalanceCurrentGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "goapi_coinbalance_current",
		Help: "Current coin balance by user from consumed events.",
	}, []string{"username"})
)

type Worker struct{}

func New() *Worker { return &Worker{} }

func (w *Worker) Handle(_ context.Context, msg kafka.Message) error {
	var event events.CoinBalanceChanged
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	coinBalanceEventsTotal.WithLabelValues(event.Username).Inc()
	delta := event.Delta
	if delta < 0 {
		delta = -delta
	}
	coinBalanceDelta.Observe(float64(delta))
	coinBalanceCurrentGauge.WithLabelValues(event.Username).Set(float64(event.Current))
	return nil
}

func HealthHandler(kafkaHealth interface{ HealthCheck(context.Context) error }) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if kafkaHealth != nil {
			if err := kafkaHealth.HealthCheck(r.Context()); err != nil {
				http.Error(w, "kafka unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
