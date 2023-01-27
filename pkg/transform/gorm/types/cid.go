package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-cid"
)

// copied from https://github.com/application-research/estuary/blob/f740b018af1d7ef6eaafb51d1c7b18a2bda0b589/util/database.go#L63

type DbCID struct {
	CID cid.Cid
}

func (dbc *DbCID) Scan(v interface{}) error {
	b, ok := v.(string)
	if !ok {
		return fmt.Errorf("dbcids must be strings")
	}

	if len(b) == 0 {
		return nil
	}

	c, err := cid.Decode(b)
	if err != nil {
		return err
	}

	dbc.CID = c
	return nil
}

func (dbc DbCID) Value() (driver.Value, error) {
	return dbc.CID.String(), nil
}

func (dbc DbCID) MarshalJSON() ([]byte, error) {
	return json.Marshal(dbc.CID.String())
}

func (dbc *DbCID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	c, err := cid.Decode(s)
	if err != nil {
		return err
	}

	dbc.CID = c
	return nil
}
