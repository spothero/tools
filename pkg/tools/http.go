package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/newrelic/go-agent"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// HTTPServerConfig contains the basic configuration necessary for running an HTTP Server
type HTTPServerConfig struct {
	Address    string
	Port       int
	Name       string
	Version    string
	AppPackage string
	GitSHA     string
	Logging    LoggingConfig
	Tracer     TracingConfig
}

type httpStatusRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func (hsr *httpStatusRecorder) WriteHeader(code int) {
	hsr.StatusCode = code
	hsr.ResponseWriter.WriteHeader(code)
}

type httpMetrics struct {
	Server   string
	Counter  *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

// HTTPMetricsRecorder defines an interface for recording prometheus metrics on HTTP requests
type HTTPMetricsRecorder interface {
	RecordHttpMetrics(w http.ResponseWriter, r *http.Request) *prometheus.Timer
}

// CLIRegistration registers required components for all HTTP Servers: Versioning, Logging, Tracing
func (c *HTTPServerConfig) CLIRegistration(flags *pflag.FlagSet, defaultPort int) {
	c.RegisterViperFlags(flags, defaultPort)

	// Logging Config
	c.Logging.AppVersion = c.Version
	c.Logging.GitSha = c.GitSHA
	c.Logging.RegisterViperFlags(flags)

	// Tracing Config
	c.Tracer.RegisterViperFlags(flags, c.Name)
}

// RunHTTPServer starts and runs a web server, waiting for a cancellation signal to exit
func (c *HTTPServerConfig) RunHTTPServer(
	preStart func(ctx context.Context, mux *http.ServeMux, server *http.Server),
	postShutdown func(ctx context.Context),
	registerMuxes func(*http.ServeMux),
) {
	c.Logging.InitializeLogger()
	closer := c.Tracer.ConfigureTracer()
	defer closer.Close()

	// Setup a context to send cancellation signals to goroutines
	ctx, cancel := context.WithCancel(context.Background())

	// Start web server
	var webWg sync.WaitGroup
	webWg.Add(1)
	go c.RunWebServer(ctx, &webWg, preStart, postShutdown, registerMuxes)

	// Setup a channel to trap interrupt signal
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals

	// Send cancellation signal to running goroutines
	Logger.Info("Received interrupt, shutting down")
	cancel()

	// Wait for web server to finish
	webWg.Wait()
	Logger.Info(fmt.Sprintf("%s service terminated", c.Name))
	// Flush any remaining logs
	Logger.Sync()
}

func makeHTTPCounter() *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP Requests received",
		},
		[]string{
			// The path recording the request
			"path",
			// The HTTP status class
			"status_class",
			// The Specific HTTP Status Code
			"status_code",
		},
	)
	prometheus.MustRegister(counter)
	return counter
}

func makeHTTPDurationHistogram() *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Total duration histogram for the HTTP request",
			// Power of 2 time - 1ms, 2ms, 4ms ... 32768ms, +Inf ms
			Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
		},
		[]string{
			// The path recording the request
			"path",
			// The HTTP status class
			"status_class",
			// The Specific HTTP Status Code
			"status_code",
		},
	)
	prometheus.MustRegister(histogram)
	return histogram
}

func initHTTPMetrics(server string) *httpMetrics {
	return &httpMetrics{
		server,
		makeHTTPCounter(),
		makeHTTPDurationHistogram(),
	}
}

func (hm *httpMetrics) recordHTTPMetrics(hsr *httpStatusRecorder, r *http.Request) *prometheus.Timer {
	return prometheus.NewTimer(prometheus.ObserverFunc(func(durationSec float64) {
		statusClass := ""
		switch {
		case hsr.StatusCode >= http.StatusInternalServerError:
			statusClass = "5xx"
		case hsr.StatusCode >= http.StatusBadRequest:
			statusClass = "4xx"
		case hsr.StatusCode >= http.StatusMultipleChoices:
			statusClass = "3xx"
		case hsr.StatusCode >= http.StatusOK:
			statusClass = "2xx"
		default:
			statusClass = "1xx"
		}
		labels := prometheus.Labels{
			"path":         r.URL.Path,
			"status_class": statusClass,
			"status_code":  strconv.Itoa(hsr.StatusCode),
		}
		hm.Counter.With(labels).Inc()
		hm.Duration.With(labels).Observe(durationSec)
	}))
}

func (hm *httpMetrics) statusCodeLogger(hsr *httpStatusRecorder, r *http.Request) func() {
	remoteAddress := zap.String("remote_address", r.RemoteAddr)
	method := zap.String("method", r.Method)
	hostname := zap.String("hostname", r.URL.Hostname())
	port := zap.String("port", r.URL.Port())
	Logger.Info("Request Received", remoteAddress, method, hostname, port)

	Logger.Debug("Request Headers", zap.Reflect("Headers", r.Header))

	return func() {
		Logger.Info(
			"Returning Response",
			remoteAddress, method, hostname, port, zap.Int("response_code", hsr.StatusCode))
	}
}

func (hm *httpMetrics) tracingHandler(hsr *httpStatusRecorder, r *http.Request) (func(), *http.Request) {
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		Logger.Debug("Failed to extract opentracing context on an incoming HTTP request.")
	}
	span, spanCtx := opentracing.StartSpanFromContext(r.Context(), r.URL.Path, ext.RPCServerOption(wireContext))
	span = span.SetTag("http.method", r.Method)
	span = span.SetTag("http.hostname", r.URL.Hostname())
	span = span.SetTag("http.port", r.URL.Port())
	span = span.SetTag("http.remote_address", r.RemoteAddr)
	return func() {
		span.SetTag("http.status_code", strconv.Itoa(hsr.StatusCode))
		// 5XX Errors are our fault -- note that this span belongs to an errored request
		if hsr.StatusCode >= http.StatusInternalServerError {
			span.SetTag("error", true)
		}
		span.Finish()
	}, r.WithContext(spanCtx)
}

// BaseHTTPMonitoringHandler is meant to be used as middleware for every request. It will:
// * Starts an opentracing span, place it in http.Request context, and close
//   close the span when the request completes
// * Starts a New Relic transaction, place it in the http.Request context, and
//   end it when the request completes
// * Capture any unhandled errors and send them to Sentry
// * Capture metrics to Prometheus for the duration of the HTTP request
func BaseHTTPMonitoringHandler(next http.Handler, serverName string) http.HandlerFunc {
	handlerMetrics := initHTTPMetrics(serverName)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default to http.StatusOK which is the golang default if the status is not set.
		wrappedWriter := &httpStatusRecorder{w, http.StatusOK}
		tracerCallback, r := handlerMetrics.tracingHandler(wrappedWriter, r)
		defer tracerCallback()
		metricsTimer := handlerMetrics.recordHTTPMetrics(wrappedWriter, r)
		defer metricsTimer.ObserveDuration()
		statusCodeLogger := handlerMetrics.statusCodeLogger(wrappedWriter, r)
		defer statusCodeLogger()
		var transaction newrelic.Transaction
		if NewRelicApp() != nil {
			transaction = NewRelicApp().StartTransaction(r.URL.Path, wrappedWriter, r)
			defer transaction.End()
		}
		type newRelicContextKey string
		r = r.WithContext(context.WithValue(r.Context(), newRelicContextKey("nrTransaction"), transaction))
		raven.RecoveryHandler(next.ServeHTTP)(wrappedWriter, r)
	})
}

// healthHandler is a simple HTTP handler that returns 200 OK
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "OK")
}

// RunWebServer starts and runs a new web server
func (c *HTTPServerConfig) RunWebServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	preStart func(ctx context.Context, mux *http.ServeMux, server *http.Server),
	postShutdown func(ctx context.Context),
	registerMuxes func(*http.ServeMux),
) {
	// Setup server and listen for requests
	defer wg.Done()
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", c.Address, c.Port),
		Handler:      BaseHTTPMonitoringHandler(mux, c.Name),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.Handler())
	// Profiling endpoints for use with go tool pprof
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	if registerMuxes != nil {
		registerMuxes(mux)
	}
	// Call any existing pre-start callback
	if preStart != nil {
		preStart(ctx, mux, server)
	}
	go func() {
		Logger.Info(fmt.Sprintf("HTTP server started on %s", server.Addr))
		if err := server.ListenAndServe(); err != nil {
			Logger.Info("HTTP server shutdown", zap.Error(err))
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	server.Shutdown(shutdown)
	// Call any existing post-shutdown callback
	if postShutdown != nil {
		postShutdown(ctx)
	}
}
