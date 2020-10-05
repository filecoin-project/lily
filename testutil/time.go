package testutil

import (
	"time"
)

// Some time functions used for working with fixed times.

var KnownTime = time.Unix(1601378000, 0).UTC()

func KnownTimeNow() time.Time {
	return KnownTime
}
