package metrics

import (
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)

var (
	TaskNS, _    = tag.NewKey("namespace")
	ConnState, _ = tag.NewKey("conn_state")
	CacheType, _ = tag.NewKey("cache_type")
	CacheOp, _   = tag.NewKey("cache_op")
)

var (
	TaskQueueLen    = stats.Int64("task_queue_len", "Length of a task queue", stats.UnitDimensionless)
	PersistDuration = stats.Float64("persist_duration_ms", "Duration of a models persist operation", stats.UnitMilliseconds)
	DBConns         = stats.Int64("db_conns", "Database connections held", stats.UnitDimensionless)
	CacheOpCount    = stats.Int64("cache_op_count", "Cache operation count", stats.UnitDimensionless)
)

var (
	TaskQueueLenView = &view.View{
		Measure:     TaskQueueLen,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{TaskNS},
	}
	PersistDurationView = &view.View{
		Measure:     PersistDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{TaskNS},
	}
	DBConnsView = &view.View{
		Measure:     DBConns,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ConnState},
	}
	CacheOpCountView = &view.View{
		Measure:     CacheOpCount,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{CacheType, CacheOp},
	}
)

var DefaultViews = append([]*view.View{
	TaskQueueLenView,
	PersistDurationView,
	DBConnsView,
	CacheOpCountView,
})

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}
