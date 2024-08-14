package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	notifications = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_controller_notifications_total",
			Help: "Number of total notification actions",
		},
	)
	notificationsFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_controller_notifications_failures_total",
			Help: "Number of failed notifications",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(notifications, notificationsFailures)
}
