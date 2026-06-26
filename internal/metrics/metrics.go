// Package metrics defines and exposes Prometheus metrics for the app.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var registry = prometheus.NewRegistry()

// TempGauge reports the last temperature (Celsius) per device.
var TempGauge = promauto.With(registry).NewGaugeVec(prometheus.GaugeOpts{
	Name: "xiaomi_temperature_celsius",
	Help: "Temperature in Celsius",
}, []string{"mac", "name"})

// HumGauge reports the last humidity (percent) per device.
var HumGauge = promauto.With(registry).NewGaugeVec(prometheus.GaugeOpts{
	Name: "xiaomi_humidity_percent",
	Help: "Humidity in percentage",
}, []string{"mac", "name"})

// Handler serves the app's metrics in Prometheus exposition format.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
