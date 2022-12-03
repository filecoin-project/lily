package commands

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-state-types/network"
	"github.com/go-pg/pg/v10"
	tablewriter "github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/storage"
)

type ModelMeta struct {
	TableName   string
	ColumnName  string
	DataType    string
	IsNullable  string
	Description string
}

type TableDesc struct {
	ObjDescription string
}

func getTableDescription(ctx context.Context, db *pg.DB, name, schema string) (string, error) {
	var description TableDesc
	_, err := db.QueryContext(ctx, &description, `
SELECT pg_catalog.obj_description(pgc.oid, 'pg_class')
FROM information_schema.tables t
         INNER JOIN pg_catalog.pg_class pgc
                    ON t.table_name = pgc.relname
WHERE t.table_type='BASE TABLE'
  AND t.table_schema=?
  AND t.table_name=?
`, schema, name)
	if err != nil {
		return "", err
	}
	return description.ObjDescription, nil
}

func getTableMetadata(ctx context.Context, db *pg.DB, table string) ([]ModelMeta, error) {
	var meta []ModelMeta
	_, err := db.QueryContext(ctx, &meta,
		`
select
    c.table_name,
    c.column_name,
    c.data_type,
    c.is_nullable,
    pgd.description
from pg_catalog.pg_statio_all_tables as st
         inner join pg_catalog.pg_description pgd on (
        pgd.objoid = st.relid
    )
         inner join information_schema.columns c on (
            pgd.objsubid   = c.ordinal_position and
            c.table_schema = st.schemaname and
            c.table_name   = st.relname
    )
where table_name = ?;
`,
		table,
	)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

var ModelDocsCmd = &cli.Command{
	Name: "gen-docs",
	Flags: FlagSet(
		dbConnectFlags,
	),
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(LilyLogFlags); err != nil {
			return fmt.Errorf("setup logging: %w", err)
		}

		ctx := cctx.Context
		db, err := storage.NewDatabase(ctx, LilyDBFlags.DB, LilyDBFlags.DBPoolSize, LilyDBFlags.Name, LilyDBFlags.DBSchema, false)
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		if err := db.Connect(ctx); err != nil {
			return err
		}
		for _, table := range tasktype.TableList {
			meta, err := getTableMetadata(ctx, db.AsORM(), table.Name)
			if err != nil {
				return err
			}

			t := tablewriter.NewWriter()
			t.AppendHeader(tablewriter.Row{"Column", "Type", "Nullable", "Description"})
			tableDescription, err := getTableDescription(ctx, db.AsORM(), table.Name, LilyDBFlags.DBSchema)
			if err != nil {
				return err
			}
			from, ok := tasktype.NetworkHeightRangeForVersion(table.NetworkVersionRange.From)
			if !ok {
				return fmt.Errorf("unsupported version: %d", table.NetworkVersionRange.From)
			}
			to, ok := tasktype.NetworkHeightRangeForVersion(table.NetworkVersionRange.To)
			if !ok {
				return fmt.Errorf("unsupported version: %d", table.NetworkVersionRange.To)
			}
			fmt.Println()
			fmt.Printf("### %s\n", table.Name)
			fmt.Printf("%s\n", tableDescription)
			fmt.Printf("* Task: `%s`\n", table.Task)
			if table.NetworkVersionRange.To == network.VersionMax {
				fmt.Printf("* Network Range: [`v%d` - `v∞`)\n", table.NetworkVersionRange.From)
			} else {
				fmt.Printf("* Network Range: [`v%d` - `v%d`]\n", table.NetworkVersionRange.From, table.NetworkVersionRange.To)
			}
			if to.To == tasktype.MaxNetworkHeight {
				fmt.Printf("* Epoch Range: [`%d` - `∞`)\n", from.From)
			} else {
				fmt.Printf("* Epoch Range: [`%d` - `%d`)\n", from.From, to.To)
			}

			if len(meta) > 0 {
				for _, m := range meta {
					t.AppendRow(tablewriter.Row{fmt.Sprintf("`%s`", m.ColumnName), fmt.Sprintf("`%s`", m.DataType), m.IsNullable, m.Description})
					t.AppendSeparator()
				}
				fmt.Println()
				fmt.Println(t.RenderMarkdown())
			}

		}
		return nil
	},
}

type ModelTypeNames struct {
	TypeName     string
	ModelName    string
	ModelComment string
	ModelFields  []string
	FieldComment map[string]string
}

func getModelTableName(t reflect.Type) string {
	modelName := Underscore(t.Name())
	// if the struct is tagged with a pg table name tag use that instead
	if f, has := t.FieldByName("tableName"); has {
		modelName = f.Tag.Get("pg")
	}
	return modelName
}

// Underscore converts "CamelCasedString" to "camel_cased_string".
func Underscore(s string) string {
	r := make([]byte, 0, len(s)+5)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if IsUpper(c) {
			if i > 0 && i+1 < len(s) && (IsLower(s[i-1]) || IsLower(s[i+1])) {
				r = append(r, '_', ToLower(c))
			} else {
				r = append(r, ToLower(c))
			}
		} else {
			r = append(r, c)
		}
	}
	return string(r)
}

func IsLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func IsUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func ToLower(c byte) byte {
	return c + 32
}
