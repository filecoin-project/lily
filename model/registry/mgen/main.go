package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"text/template"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/registry"

	// imported for the side effect of calling each packages init method
	_ "github.com/filecoin-project/sentinel-visor/model/actors/common"
	_ "github.com/filecoin-project/sentinel-visor/model/actors/reward"
	_ "github.com/filecoin-project/sentinel-visor/model/blocks"
	_ "github.com/filecoin-project/sentinel-visor/model/chain"
	_ "github.com/filecoin-project/sentinel-visor/model/derived"
	_ "github.com/filecoin-project/sentinel-visor/model/messages"
	_ "github.com/filecoin-project/sentinel-visor/model/msapprovals"
	// refactor
	_ "github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner"
)

func main() {
	if err := generateModelNames("model/registry/mgen"); err != nil {
		panic(err)
	}
}

func generateModelNames(dir string) error {
	mf, err := ioutil.ReadFile(filepath.Join(dir, "gen.go.template"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // skip it.
		}
		return xerrors.Errorf("loading mode gen template: %w", err)
	}
	modelPkgsMap := make(map[string]struct{})
	modelMap := make(map[string]model.Persistable)
	modelNames := make([]string, len(registry.ModelRegistry.RegisteredModels()))
	for i, modelType := range registry.ModelRegistry.RegisteredModels() {
		modelName := pg.Model(modelType).TableModel().Table().ModelName
		modelMap[modelName] = modelType
		modelNames[i] = modelName
		modelPkgsMap[reflect.TypeOf(modelType).Elem().PkgPath()] = struct{}{}
	}
	var modelPkgs []string
	for k := range modelPkgsMap {
		modelPkgs = append(modelPkgs, k)
	}

	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"getModelType": func(name string) reflect.Type { return reflect.TypeOf(modelMap[name]).Elem() },
	}).Parse(string(mf)))

	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]interface{}{
		"modelNames": modelNames,
		"modelPkgs":  modelPkgs,
	})
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "../registered/gen.go"), b.Bytes(), 0o666); err != nil {
		return err
	}
	return nil
}
