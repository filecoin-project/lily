package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	TaskNS, _ = tag.NewKey("namespace")
)

var (
	TaskQueueLen = stats.Int64("task_queue_len", "Length of a task queue", stats.UnitDimensionless)
)

var (
	TaskQueueLenView = &view.View{
		Measure: TaskQueueLen,
		Aggregation: view.Sum(),
		TagKeys: []tag.Key{TaskNS},
	}
)

var DefaultViews = append([]*view.View{
	TaskQueueLenView,
})