package storage

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
)

var models = []interface{}{
	(*blocks.BlockHeader)(nil),
	(*blocks.BlockSynced)(nil),
	(*blocks.BlockParent)(nil),
	(*blocks.DrandEntrie)(nil),
	(*blocks.DrandBlockEntrie)(nil),
	(*miner.MinerPower)(nil),
	(*miner.MinerState)(nil),
	(*miner.MinerSectorInfo)(nil),
	(*miner.MinerPreCommitInfo)(nil),
}

func NewDatabase(ctx context.Context, url string) (*Database, error) {
	opt, err := pg.ParseURL(url)
	if err != nil {
		return nil, xerrors.Errorf("parse database URL: %w", err)
	}

	db := pg.Connect(opt)
	db = db.WithContext(ctx)

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

func (d *Database) CollectAndMarkBlocksAsProcessing(ctx context.Context, batch int) (blocks.BlocksSynced, error) {
	var blks blocks.BlocksSynced
	processedAt := time.Now()
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if _, err := tx.QueryContext(ctx, &blks,
			`with toProcess as (
					select cid, height, rank() over (order by height) as rnk
					from blocks_synced
					where completed_at is null and
					processed_at is null and
					height > 0
				)
				select cid
				from toProcess
				where rnk <= ?
				for update skip locked`, // ensure that only a single process can select and update blocks as processing.
			batch,
		); err != nil {
			return err
		}
		for _, blk := range blks {
			if _, err := tx.ModelContext(ctx, blk).Set("processed_at = ?", processedAt).
				WherePK().
				Update(); err != nil {
				return xerrors.Errorf("marking block as processed: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return blks, nil
}

func (d *Database) MarkBlocksAsProcessed(ctx context.Context, blks blocks.BlocksSynced) error {
	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		completedAt := time.Now()
		for _, blk := range blks {
			if _, err := tx.ModelContext(ctx, &blk).Set("completed_at = ?", completedAt).
				WherePK().
				Update(); err != nil {
				return err
			}
		}
		return nil
	})
}
