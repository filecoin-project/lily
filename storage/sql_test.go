package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-pg/pg/v10/orm"
	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/sentinel-visor/testutil"
)

func TestSchemaIsCurrent(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx := context.Background()

	d, err := NewDatabase(ctx, testutil.Database(), 10)
	if !assert.NoError(t, err, "connecting to database") {
		return
	}

	db, err := connect(ctx, d.opt)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	for _, model := range models {
		t.Run(fmt.Sprintf("%T", model), func(t *testing.T) {
			q := db.Model(model)
			err := verifyModel(ctx, db, q.TableModel().Table())
			if err != nil {
				t.Errorf("%v", err)
				ctq := orm.NewCreateTableQuery(q, &orm.CreateTableOptions{IfNotExists: true})
				t.Logf("Expect %s", ctq.String())
			}
		})
	}
}
