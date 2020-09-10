package storage

import (
	"context"

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
		Where("processes_at is null").
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
