package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/filecoin-project/go-state-types/big"
)

type DbBigInt struct {
	BigInt big.Int
}

func (dbb *DbBigInt) Scan(v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("DbBigInts must be strings")
	}

	bi, err := big.FromString(s)
	if err != nil {
		return err
	}

	dbb.BigInt = bi
	return nil
}

func (dbb DbBigInt) Value() (driver.Value, error) {
	return dbb.BigInt.String(), nil
}

func (DbBigInt) GormDataType() string {
	return "bigint"
}
