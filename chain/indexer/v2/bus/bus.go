package bus

import (
	evntbus "github.com/mustafaturan/bus/v3"
	"github.com/mustafaturan/monoton/v2"
	"github.com/mustafaturan/monoton/v2/sequencer"
)

func NewBus() (*Bus, error) {
	// configure id generator (it doesn't have to be monoton)
	node := uint64(1)
	initialTime := uint64(1577865600000) // set 2020-01-01 PST as initial time
	m, err := monoton.New(sequencer.NewMillisecond(), node, initialTime)
	if err != nil {
		return nil, err
	}

	// init an id generator
	var idGenerator evntbus.Next = m.Next

	// create a new bus instance
	b, err := evntbus.NewBus(idGenerator)
	if err != nil {
		return nil, err
	}
	return &Bus{
		Bus: b,
	}, nil
}

type Bus struct {
	Bus *evntbus.Bus
}
