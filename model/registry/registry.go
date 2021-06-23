package registry

import (
	"github.com/filecoin-project/sentinel-visor/model"
)

type Registry struct {
	Models model.PersistableList
}

func NewRegistry() *Registry {
	return &Registry{
		Models: model.PersistableList{},
	}
}

func (r *Registry) Register(m model.Persistable) {
	r.Models = append(r.Models, m)
}

func (r *Registry) RegisteredModels() model.PersistableList {
	return r.Models
}

var ModelRegistry = NewRegistry()
