package storage

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/visor/model/actors"
	"github.com/filecoin-project/visor/model/blocks"
)

var models = []interface{}{
	(*blocks.BlockHeader)(nil),
	(*blocks.BlockSynced)(nil),
	(*blocks.BlockParent)(nil),
	(*blocks.DrandEntrie)(nil),
	(*blocks.DrandBlockEntrie)(nil),
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

func (d *Database) UnprocessedIndexedBlocks(ctx context.Context, maxHeight, limit int) (blocks.BlocksSynced, error) {
	var blkSynced blocks.BlocksSynced
	if err := d.DB.ModelContext(ctx, &blkSynced).
		Where("height <= ?", maxHeight).
		Where("processed_at is null").
		Order("height desc").
		Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) MostRecentProcessedBlock(ctx context.Context) (*blocks.BlockSynced, error) {
	blkSynced := &blocks.BlockSynced{}
	if err := d.DB.ModelContext(ctx, blkSynced).
		Order("height desc").
		Limit(1).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) CollectBlocksForProcessing(ctx context.Context, batch int) (blocks.BlocksSynced, error) {
	var blks blocks.BlocksSynced
	if _, err := d.DB.QueryContext(ctx, &blks,
		`with toProcess as (
					select cid, height, rank() over (order by height) as rnk
					from blocks_synced
					where completed_at is null and
					processed_at is null and
					height > 0
				)
				select cid
				from toProcess
				where rnk <= ?`,
		batch,
	); err != nil {
		return nil, xerrors.Errorf("collecting blocks for processing: %w", err)
	}
	return blks, nil
}

func (d *Database) MarkBlocksAsProcessing(ctx context.Context, blks blocks.BlocksSynced) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}

	processedAt := time.Now()
	for _, blk := range blks {
		if _, err := tx.ModelContext(ctx, blk).Set("processed_at = ?", processedAt).
			WherePK().
			Update(); err != nil {
			return xerrors.Errorf("marking block as processed: %w", err)
		}
	}
	return tx.CommitContext(ctx)
}

func (d *Database) MarkBlocksAsProcessed(ctx context.Context, blks blocks.BlocksSynced) error {
	tx, err := d.DB.BeginContext(ctx)
	if err != nil {
		return err
	}

	completedAt := time.Now()
	for _, blk := range blks {
		if _, err := tx.ModelContext(ctx, &blk).Set("completed_at = ?", completedAt).
			WherePK().
			Update(); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)
}
