package lizard

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/go-pg/pg/v10"
)

type LogDebugHook struct {
}

func (l LogDebugHook) BeforeQuery(ctx context.Context, evt *pg.QueryEvent) (context.Context, error) {
	q, err := evt.FormattedQuery()
	if err != nil {
		return nil, err
	}

	if evt.Err != nil {
		log.Errorf("%s executing a query:%s", evt.Err, q)
	}
	fmt.Println(string(q))

	return ctx, nil
}

func (l LogDebugHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	log.Infow("Executed", "duration", time.Since(event.StartTime), "rows_returned", event.Result.RowsReturned())
	return nil
}

var nameAndVersion = regexp.MustCompile(`^(.+?)\+`)

// NormalizeAgent attempts to normalize an agent string to a software name and major/minor version
func NormalizeAgent(agent string) string {
	m := nameAndVersion.FindStringSubmatch(agent)
	if len(m) > 1 {
		return m[1]
	}

	return agent
}
