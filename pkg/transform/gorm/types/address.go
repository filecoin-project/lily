package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/filecoin-project/go-address"
)

//copied from https://github.com/application-research/estuary/blob/f740b018af1d7ef6eaafb51d1c7b18a2bda0b589/util/database.go#L63

type DbAddr struct {
	Addr address.Address
}

func (dba *DbAddr) Scan(v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("DbAddrs must be strings")
	}

	addr, err := address.NewFromString(s)
	if err != nil {
		return err
	}

	dba.Addr = addr
	return nil
}

func (dba DbAddr) Value() (driver.Value, error) {
	return dba.Addr.String(), nil
}
