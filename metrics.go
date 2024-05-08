package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	googleDriveUploadDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "dpr_google_drive_upload_duration",
			Help: "Histogram of the duration of Google Drive Upload Requests.",
		},
		[]string{},
	)

	batchProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "dpr_batch_processing_time",
			Help: "Histogram of the duration of Discord Message Batch Processing.",
		},
		[]string{},
	)

	messagesChecked = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dpr_messages_checked",
			Help: "# of messages scanned",
		},
	)

	uploadedFiles = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dpr_uploaded_files",
			Help: "# of uploaded files",
		},
	)

	lastRunSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dpr_success",
			Help: "The application records a 1 on successful exit",
		},
		[]string{},
	)
)

func initMetrics() {
	METRICS_HTTP_PORT := "8889"
	if os.Getenv("METRICS_HTTP_PORT") != "" {
		METRICS_HTTP_PORT = os.Getenv("METRICS_HTTP_PORT")
	}

	prometheus.MustRegister(googleDriveUploadDuration)
	prometheus.MustRegister(batchProcessingTime)
	prometheus.MustRegister(messagesChecked)
	prometheus.MustRegister(lastRunSuccess)
	prometheus.MustRegister(uploadedFiles)

	// Expose Prometheus metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":"+METRICS_HTTP_PORT, nil)
		if err != nil {
			log.Fatal("Failed to start Prometheus metrics server: ", err)
		}
	}()

}
