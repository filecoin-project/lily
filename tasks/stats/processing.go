package stats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

var statsInsert = `INSERT INTO visor_processing_stats SELECT date_trunc('minute', NOW()), measure, value FROM ( %s ) stats ON CONFLICT DO NOTHING;`

var statsTipsetsTemplate = `
-- total number of tipsets that have been discovered for processing
SELECT 'tipsets_%[1]s_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_tipsets

UNION

-- total number of tipsets that have been processed
SELECT 'tipsets_%[1]s_completed_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NOT NULL

UNION

-- total number of tipsets that have been processed but reported an error
SELECT 'tipsets_%[1]s_errors_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NOT NULL

UNION

-- total number of tipsets that are currently being processed
SELECT 'tipsets_%[1]s_claimed_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_tipsets WHERE %[1]s_claimed_until IS NOT NULL

UNION

-- highest epoch that has been processed
SELECT 'tipsets_%[1]s_completed_height_max' AS measure, COALESCE(max(height),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NULL

UNION

-- highest epoch that has not been processed
SELECT 'tipsets_%[1]s_incomplete_height_max' AS measure, COALESCE(max(height),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NULL
`

var statsMessagesTemplate = `
-- total number of messages that have been discovered for processing
SELECT 'messages_%[1]s_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_messages

UNION

-- total number of messages that have been processed
SELECT 'messages_%[1]s_completed_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NOT NULL

UNION

-- total number of messages that have been processed but reported an error
SELECT 'messages_%[1]s_errors_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NOT NULL

UNION

-- total number of messages that are currently being processed
SELECT 'messages_%[1]s_claimed_count' AS measure, COALESCE(count(*),0) AS value FROM visor_processing_messages WHERE %[1]s_claimed_until IS NOT NULL

UNION

-- highest epoch that has been processed
SELECT 'messages_%[1]s_completed_height_max' AS measure, COALESCE(max(height),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NULL

UNION

-- highest epoch that has not been processed
SELECT 'messages_%[1]s_incomplete_height_max' AS measure, COALESCE(max(height),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NULL
`

var statsActors = `
-- total number of actors of each type that have been discovered for processing
SELECT concat('actors_', code, '_count') AS measure, COALESCE(count(*),0) AS value FROM visor_processing_actors GROUP BY code

UNION

-- total number of actors of each type that have been processed
SELECT concat('actors_', code, '_completed_count') AS measure, COALESCE(count(*),0) AS value FROM visor_processing_actors WHERE completed_at IS NOT NULL GROUP BY code

UNION

-- total number of actors of each type that have been processed but reported an error
SELECT concat('actors_', code, '_errors_count') AS measure, COALESCE(count(*),0) AS value FROM visor_processing_actors WHERE completed_at IS NOT NULL AND errors_detected IS NOT NULL GROUP BY code

UNION

-- total number of actors of each type that have are currently being processed
SELECT concat('actors_', code, '_claimed_count') AS measure, COALESCE(count(*),0) AS value FROM visor_processing_actors WHERE claimed_until IS NOT NULL GROUP BY code

UNION

-- highest epoch that has been processed
SELECT concat('actors_', code, '_completed_height_max') AS measure, COALESCE(max(height),0) AS value FROM visor_processing_actors WHERE completed_at IS NOT NULL AND errors_detected IS NULL GROUP BY code

UNION

-- highest epoch that has not been processed
SELECT concat('actors_', code, '_incomplete_height_max') AS measure, COALESCE(max(height),0) AS value FROM visor_processing_actors WHERE completed_at IS NULL GROUP BY code
`

func NewProcessingStatsRefresher(d *storage.Database, refreshRate time.Duration) *ProcessingStatsRefresher {
	return &ProcessingStatsRefresher{
		db:          d,
		refreshRate: refreshRate,
	}
}

// ProcessingStatsRefresher is a task which periodically collects summaries of processing tables used by visor
type ProcessingStatsRefresher struct {
	db          *storage.Database
	refreshRate time.Duration
}

// Run starts regularly refreshing until context is done or an error occurs
func (r *ProcessingStatsRefresher) Run(ctx context.Context) error {
	if r.refreshRate == 0 {
		return nil
	}
	return wait.RepeatUntil(ctx, r.refreshRate, r.collectStats)
}

func (r *ProcessingStatsRefresher) collectStats(ctx context.Context) (bool, error) {
	subQueries := []string{statsActors}

	tipsetTaskTypes := []string{"message", "statechange", "economics"}

	for _, taskType := range tipsetTaskTypes {
		subQueries = append(subQueries, fmt.Sprintf(statsTipsetsTemplate, taskType))
	}

	messageTaskTypes := []string{"gas_outputs"}

	for _, taskType := range messageTaskTypes {
		subQueries = append(subQueries, fmt.Sprintf(statsMessagesTemplate, taskType))
	}

	subQuery := strings.Join(subQueries, " UNION ")

	_, err := r.db.DB.ExecContext(ctx, fmt.Sprintf(statsInsert, subQuery))
	if err != nil {
		return true, xerrors.Errorf("refresh: %w", err)
	}

	return false, nil
}
