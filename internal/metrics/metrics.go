package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Transaction metrics
	transactionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transactions_total",
			Help: "Total number of transactions",
		},
		[]string{"type", "status"},
	)

	transactionAmount = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transaction_amount",
			Help:    "Transaction amounts",
			Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000},
		},
		[]string{"type"},
	)

	transactionProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transaction_processing_duration_seconds",
			Help:    "Transaction processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// User metrics
	usersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of users",
		},
	)

	activeUsersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users_total",
			Help: "Number of active users",
		},
	)

	// Balance metrics
	totalBalance = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "total_balance",
			Help: "Total balance across all users",
		},
	)

	userBalances = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "user_balances",
			Help:    "Distribution of user balances",
			Buckets: []float64{0, 10, 50, 100, 500, 1000, 5000, 10000, 50000},
		},
		[]string{},
	)

	// Worker pool metrics
	workerPoolJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_pool_jobs_total",
			Help: "Total number of jobs processed by worker pool",
		},
		[]string{"status"},
	)

	workerPoolActiveJobs = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "worker_pool_active_jobs",
			Help: "Number of jobs currently being processed",
		},
	)

	workerPoolQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "worker_pool_queue_size",
			Help: "Current size of the worker pool queue",
		},
	)

	// Database metrics
	databaseConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of database connections",
		},
		[]string{"state"},
	)

	databaseQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	databaseQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// Cache metrics
	cacheOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"},
	)

	cacheHitRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_hit_ratio",
			Help: "Cache hit ratio",
		},
		[]string{"cache_type"},
	)
)

// Init initializes the metrics
func Init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		transactionsTotal,
		transactionAmount,
		transactionProcessingDuration,
		usersTotal,
		activeUsersTotal,
		totalBalance,
		userBalances,
		workerPoolJobsTotal,
		workerPoolActiveJobs,
		workerPoolQueueSize,
		databaseConnections,
		databaseQueriesTotal,
		databaseQueryDuration,
		cacheOperationsTotal,
		cacheHitRatio,
	)
}

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// HTTP Metrics
func RecordHTTPRequest(method, endpoint, status string) {
	httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

func RecordHTTPDuration(method, endpoint string, duration float64) {
	httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// Transaction Metrics
func RecordTransaction(txType, status string) {
	transactionsTotal.WithLabelValues(txType, status).Inc()
}

func RecordTransactionAmount(txType string, amount float64) {
	transactionAmount.WithLabelValues(txType).Observe(amount)
}

func RecordTransactionProcessingDuration(txType string, duration float64) {
	transactionProcessingDuration.WithLabelValues(txType).Observe(duration)
}

// User Metrics
func SetUsersTotal(count float64) {
	usersTotal.Set(count)
}

func SetActiveUsersTotal(count float64) {
	activeUsersTotal.Set(count)
}

// Balance Metrics
func SetTotalBalance(balance float64) {
	totalBalance.Set(balance)
}

func RecordUserBalance(balance float64) {
	userBalances.WithLabelValues().Observe(balance)
}

// Worker Pool Metrics
func RecordWorkerPoolJob(status string) {
	workerPoolJobsTotal.WithLabelValues(status).Inc()
}

func SetWorkerPoolActiveJobs(count float64) {
	workerPoolActiveJobs.Set(count)
}

func SetWorkerPoolQueueSize(size float64) {
	workerPoolQueueSize.Set(size)
}

// Database Metrics
func SetDatabaseConnections(state string, count float64) {
	databaseConnections.WithLabelValues(state).Set(count)
}

func RecordDatabaseQuery(operation, table string) {
	databaseQueriesTotal.WithLabelValues(operation, table).Inc()
}

func RecordDatabaseQueryDuration(operation, table string, duration float64) {
	databaseQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

// Cache Metrics
func RecordCacheOperation(operation, result string) {
	cacheOperationsTotal.WithLabelValues(operation, result).Inc()
}

func SetCacheHitRatio(cacheType string, ratio float64) {
	cacheHitRatio.WithLabelValues(cacheType).Set(ratio)
}
