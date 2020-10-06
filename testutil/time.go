package testutil

import (
	"time"

	"github.com/raulk/clock"
)

// Some time functions used for working with fixed times.

var KnownTime = time.Unix(1601378000, 0).UTC()

func NewMockClock() *clock.Mock {
	m := clock.NewMock()
	m.Set(KnownTime)
	return m
}
