package storage

import (
	"context"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/visor/model"
	"github.com/filecoin-project/visor/model/actors"
	"github.com/filecoin-project/visor/model/blocks"
	"github.com/filecoin-project/visor/model/tasks"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
	"time"
)

var models = []interface{}{
	(*blocks.BlockHeader)(nil),
	(*tasks.BlockProcessTask)(nil),
	(*actors.MinerPower)(nil),
	(*actors.MinerState)(nil),
}

func NewDatabase(ctx context.Context, url string) (*Database, error) {
	opt, err := pg.ParseURL(url)
	if err != nil {
		return nil, xerrors.Errorf("parse database URL: %w", err)
	}

	db := pg.Connect(opt)
	// Check if connection credentials are valid and PostgreSQL is up and running.
	if err := db.Ping(ctx); err != nil {
		return nil, xerrors.Errorf("ping database: %w", err)
	}

	return &Database{DB: db}, nil
}

type Database struct {
	DB *pg.DB
}

func (d *Database) CreateSchema() error {
	for _, model := range models {
		if err := d.DB.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		}); err != nil {
			return xerrors.Errorf("creating table: %w", err)
		}
	}
	return nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) StoreBlockHeaders(ctx context.Context, headers map[cid.Cid]*types.BlockHeader) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}

	for _, header := range headers {
		if _, err := tx.ModelContext(ctx, blocks.NewBlockHeader(header)).
			OnConflict("do nothing").
			Insert(); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)
}

func (d *Database) StoreMinerPower(ctx context.Context, res model.MinerStateResult) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ModelContext(ctx, actors.NewMinerPowerModel(res)).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return tx.CommitContext(ctx)

}

func (d *Database) StoreMinerState(ctx context.Context, res model.MinerStateResult) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ModelContext(ctx, actors.NewMinerStateModel(res)).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return tx.CommitContext(ctx)
}

func (d *Database) CreateBlockProcessTask(ctx context.Context, headers map[cid.Cid]*types.BlockHeader, createdAt time.Time) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}

	for _, header := range headers {
		if _, err := tx.ModelContext(ctx, tasks.NewBlockProcessTask(header, createdAt)).
			OnConflict("do nothing").
			Insert(); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)
}

func (d *Database) CompletedBlockProcessTasks(ctx context.Context, maxHeight, limit int) ([]tasks.BlockProcessTask, error) {
	var blkTasks []tasks.BlockProcessTask
	if err := d.DB.ModelContext(ctx, &blkTasks).
		Where("height <= ?", maxHeight).
		Where("completed_at is not null").
		Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	return blkTasks, nil
}

func (d *Database) IncompleteBlockProcessTasks(ctx context.Context, maxHeight, limit int) ([]tasks.BlockProcessTask, error) {
	var blkTasks []tasks.BlockProcessTask
	if err := d.DB.ModelContext(ctx, &blkTasks).
		Where("height <= ?", maxHeight).
		Where("completed_at is null").
		Order("height desc").
		Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	return blkTasks, nil
}

func (d *Database) MostRecentCompletedBlockProcessTask(ctx context.Context) (*tasks.BlockProcessTask, error) {
	blkTasks := new(tasks.BlockProcessTask)
	if err := d.DB.ModelContext(ctx, blkTasks).
		//Where("completed_at is not null").
		Order("height desc").
		Limit(1).
		Select(); err != nil {
		return nil, err
	}
	return blkTasks, nil
}

func (d *Database) GetBlocksForFeeder(ctx context.Context, limit int) ([]tasks.BlockProcessTask, error) {
	// TODO I have no idea if this works (I trust the query, I don't understand the ORM)
	// and don't have the time to spend making it perfect.
	// TODO use OUTPUT to select and update these at the same time (update them to show they are being processed)
	var blkTasks []tasks.BlockProcessTask
	if _, err := d.DB.QueryContext(ctx,
		&blkTasks,
		`with toProcess as (
					select cid, height, rank() over (order by height) as rnk
					from block_process_tasks
					where completed_at is null and
					attempted_at is null and
					height > 0
				)
				select cid
				from toProcess
				where rnk <= ?`,
		limit,
	); err != nil {
		return nil, err
	}
	return blkTasks, nil

}

func (d *Database) MarkBlocksAsProcessing(ctx context.Context, tasks []tasks.BlockProcessTask, start time.Time) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if _, err := tx.ModelContext(ctx, &t).Set("attempted_at = ?", start).
			WherePK().
			Update(); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)
}

func (d *Database) MarkBlocksAsComplete(ctx context.Context, tasks []tasks.BlockProcessTask, complete time.Time) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if _, err := tx.ModelContext(ctx, &t).Set("completed_at = ?", complete).
			WherePK().
			Update(); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)

}
