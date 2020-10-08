package metrics

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)

var (
	Error, _ = tag.NewKey("error")
	TaskType, _ = tag.NewKey("task")
	ConnState, _ = tag.NewKey("conn_state")
	State, _ = tag.NewKey("state")
)

var (
	ProcessingDuration = stats.Float64("processing_duration_ms", "Time taken to process a single item", stats.UnitMilliseconds)
	PersistDuration = stats.Float64("persist_duration_ms", "Duration of a models persist operation", stats.UnitMilliseconds)
	DBConns = stats.Int64("db_conns", "Database connections held", stats.UnitDimensionless)
	HistoricalIndexerHeight = stats.Int64("historical_sync_height", "Sync height of the historical indexer", stats.UnitDimensionless)
)

var (
	ProcessingDurationView = &view.View{
		Measure: ProcessingDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys: []tag.Key{TaskType},
	}
	PersistDurationView = &view.View{
		Measure: PersistDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys: []tag.Key{TaskType},
	}
	DBConnsView = &view.View{
		Measure: DBConns,
		Aggregation: view.Count(),
		TagKeys: []tag.Key{ConnState},
	}
	HistoricalIndexerHeightView = &view.View{
		Measure: HistoricalIndexerHeight,
		Aggregation: view.LastValue(),
		TagKeys: []tag.Key{State},
	}
)

var DefaultViews = append([]*view.View{
	ProcessingDurationView,
	PersistDurationView,
	DBConnsView,
	HistoricalIndexerHeightView,
})

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

// Timer is a function stopwatch, calling it starts the timer,
// calling the returned function will record the duration.
func Timer(ctx context.Context, m *stats.Float64Measure) func() {
	start := time.Now()
	return func() {
		stats.Record(ctx, m.M(SinceInMilliseconds(start)))
	}
}