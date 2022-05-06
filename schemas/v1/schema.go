package v1

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/go-pg/migrations/v8"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/schemas"
)

const MajorVersion = 1

func init() {
	schemas.RegisterSchema(MajorVersion)
}

func GetBase(cfg schemas.Config) (string, error) {
	tmpl, err := template.New("base").Funcs(schemaTemplateFuncMap).Parse(BaseTemplate)
	if err != nil {
		return "", fmt.Errorf("parse base template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("execute base template: %w", err)
	}
	return buf.String(), nil
}

func GetPatches(cfg schemas.Config) (*migrations.Collection, error) {
	return patches.Collection(cfg)
}

func Version() model.Version {
	return model.Version{
		Major: MajorVersion,
		Patch: len(patches.pm),
	}
}

var patches = NewPatchList()

type patch struct {
	seq  int
	tmpl *template.Template
}

type patchList struct {
	pm map[int]patch
}

func NewPatchList() patchList {
	return patchList{map[int]patch{}}
}

// Register adds a patch to the patch list. This should be called in an init function.
func (pl *patchList) Register(seq int, text string) {
	if seq <= 0 {
		panic(fmt.Sprintf("invalid patch number: %d", seq))
	}

	if _, exists := pl.pm[seq]; exists {
		panic(fmt.Sprintf("duplicate patch registered: %d", seq))
	}

	tmpl, err := template.New("patch").Funcs(schemaTemplateFuncMap).Parse(text)
	if err != nil {
		panic(fmt.Sprintf("parse patch template: %v", err))
	}

	pl.pm[seq] = patch{
		seq:  seq,
		tmpl: tmpl,
	}
}

func (pl *patchList) Collection(cfg schemas.Config) (*migrations.Collection, error) {
	// Check patch list is consistent with no gaps
	count := len(pl.pm)

	// patch 0 must not exist - it's the base schema by definition
	if _, exists := pl.pm[0]; exists {
		return nil, fmt.Errorf("found patch 0, which should not exist")
	}

	// index from 1 since schema seq 0 is the base and not in `pm`
	for i := 1; i <= count; i++ {
		if _, exists := pl.pm[i]; !exists {
			return nil, fmt.Errorf("missing patch %d", i)
		}
	}

	migs := make([]*migrations.Migration, 0, count)
	for i := 1; i <= count; i++ {
		p := pl.pm[i]

		var buf strings.Builder
		if err := p.tmpl.Execute(&buf, cfg); err != nil {
			return nil, fmt.Errorf("execute patch template: %w", err)
		}
		sql := buf.String()

		migs = append(migs, &migrations.Migration{
			Version: int64(i),
			UpTx:    true,
			Up: func(db migrations.DB) error {
				if _, err := db.Exec(sql); err != nil {
					return err
				}
				return nil
			},
		})
	}

	coll := migrations.NewCollection(migs...)
	coll.SetTableName(cfg.SchemaName + ".gopg_migrations")
	return coll, nil
}

var schemaTemplateFuncMap = template.FuncMap{
	"default": func(def interface{}, value interface{}) interface{} {
		if isEmpty(value) {
			return def
		}
		return value
	},
}

func isEmpty(val interface{}) bool {
	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		return false
	default:
		return v.IsNil()
	}
}
