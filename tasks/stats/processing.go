package stats

import (
	"context"
	"fmt"
	"strings"
	"time"

	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

var statsInsert = `INSERT INTO visor_processing_stats SELECT date_trunc('minute', NOW()), measure, tag, value FROM ( %s ) stats ON CONFLICT DO NOTHING;`

var statsTipsetsTemplate = `
-- total number of tipsets that have been discovered for processing
SELECT 'count' AS measure, 'tipsets_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_tipsets

UNION

-- total number of tipsets that have been processed (includes errors)
SELECT 'completed_count' AS measure, 'tipsets_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NOT NULL

UNION

-- total number of tipsets that have been processed but reported an error
SELECT 'errors_count' AS measure, 'tipsets_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NOT NULL

UNION

-- total number of tipsets that are currently being processed
SELECT 'claimed_count' AS measure, 'tipsets_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_tipsets WHERE %[1]s_claimed_until IS NOT NULL

UNION

-- highest epoch that has been processed successfully
SELECT 'completed_height_max' AS measure, 'tipsets_%[1]s' AS tag, COALESCE(max(height),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NULL

UNION

-- highest epoch that has not been processed
SELECT 'incomplete_height_max' AS measure, 'tipsets_%[1]s' AS tag, COALESCE(max(height),0) AS value FROM visor_processing_tipsets WHERE %[1]s_completed_at IS NULL
`

var statsMessagesTemplate = `
-- total number of messages that have been discovered for processing
SELECT 'count' AS measure, 'messages_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_messages

UNION

-- total number of messages that have been processed (includes errors)
SELECT 'completed_count' AS measure, 'messages_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NOT NULL

UNION

-- total number of messages that have been processed but reported an error
SELECT 'errors_count' AS measure, 'messages_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NOT NULL

UNION

-- total number of messages that are currently being processed
SELECT 'claimed_count' AS measure, 'messages_%[1]s' AS tag, COALESCE(count(*),0) AS value FROM visor_processing_messages WHERE %[1]s_claimed_until IS NOT NULL

UNION

-- highest epoch that has been processed successfully
SELECT 'completed_height_max' AS measure, 'messages_%[1]s' AS tag, COALESCE(max(height),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NOT NULL AND %[1]s_errors_detected IS NULL

UNION

-- highest epoch that has not been processed
SELECT 'incomplete_height_max' AS measure, 'messages_%[1]s' AS tag, COALESCE(max(height),0) AS value FROM visor_processing_messages WHERE %[1]s_completed_at IS NULL
`

var statsActors = `
-- total number of actors of each type that have been discovered for processing
SELECT 'count' AS measure, %[1]s as tag, COALESCE(count(*),0) AS value FROM visor_processing_actors GROUP BY code

UNION

-- total number of actors of each type that have been processed (includes errors)
SELECT 'completed_count' AS measure, %[1]s as tag, COALESCE(count(*),0) AS value FROM visor_processing_actors WHERE completed_at IS NOT NULL GROUP BY code

UNION

-- total number of actors of each type that have been processed but reported an error
SELECT 'errors_count' AS measure, %[1]s as tag, COALESCE(count(*),0) AS value FROM visor_processing_actors WHERE completed_at IS NOT NULL AND errors_detected IS NOT NULL GROUP BY code

UNION

-- total number of actors of each type that have are currently being processed
SELECT 'claimed_count' AS measure, %[1]s as tag, COALESCE(count(*),0) AS value FROM visor_processing_actors WHERE claimed_until IS NOT NULL GROUP BY code

UNION

-- highest epoch that has been processed successfully
SELECT 'completed_height_max' AS measure, %[1]s as tag, COALESCE(max(height),0) AS value FROM visor_processing_actors WHERE completed_at IS NOT NULL AND errors_detected IS NULL GROUP BY code

UNION

-- highest epoch that has not been processed
SELECT 'incomplete_height_max' AS measure, %[1]s as tag, COALESCE(max(height),0) AS value FROM visor_processing_actors WHERE completed_at IS NULL GROUP BY code
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
	subQueries := []string{fmt.Sprintf(statsActors, actorCodeCase)}

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

var actorCodeTags = map[string]string{
	sa0builtin.SystemActorCodeID.String():           "actor_system_1",
	sa0builtin.InitActorCodeID.String():             "actor_init_1",
	sa0builtin.CronActorCodeID.String():             "actor_cron_1",
	sa0builtin.StoragePowerActorCodeID.String():     "actor_storagepower_1",
	sa0builtin.StorageMinerActorCodeID.String():     "actor_storageminer_1",
	sa0builtin.StorageMarketActorCodeID.String():    "actor_storagemarker_1",
	sa0builtin.PaymentChannelActorCodeID.String():   "actor_paymentchannel_1",
	sa0builtin.RewardActorCodeID.String():           "actor_reward_1",
	sa0builtin.VerifiedRegistryActorCodeID.String(): "actor_verifiedregistry_1",
	sa0builtin.AccountActorCodeID.String():          "actor_account_1",
	sa0builtin.MultisigActorCodeID.String():         "actor_multisig_1",
	sa2builtin.SystemActorCodeID.String():           "actor_system_2",
	sa2builtin.InitActorCodeID.String():             "actor_init_2",
	sa2builtin.CronActorCodeID.String():             "actor_cron_2",
	sa2builtin.StoragePowerActorCodeID.String():     "actor_storagepower_2",
	sa2builtin.StorageMinerActorCodeID.String():     "actor_storageminer_2",
	sa2builtin.StorageMarketActorCodeID.String():    "actor_storagemarket_2",
	sa2builtin.PaymentChannelActorCodeID.String():   "actor_paymentchannel_2",
	sa2builtin.RewardActorCodeID.String():           "actor_reward_2",
	sa2builtin.VerifiedRegistryActorCodeID.String(): "actor_verifiedregistry_2",
	sa2builtin.AccountActorCodeID.String():          "actor_account_2",
	sa2builtin.MultisigActorCodeID.String():         "actor_multisig_2",
}

// actorCodeCase is a SQL CASE statement that replaces builtin actor codes with nice names
var actorCodeCase string

func init() {
	var cases []string
	for code, tag := range actorCodeTags {
		cases = append(cases, fmt.Sprintf("WHEN code='%s' THEN '%s'", code, tag))
	}

	actorCodeCase = fmt.Sprintf("CASE %s ELSE concat('actor_', code) END", strings.Join(cases, " "))
}
