package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
)

type DbToken struct {
	Token abi.TokenAmount
}

func (dbt *DbToken) Scan(v interface{}) error {
	switch v.(type) {
	case string:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("DbToken must be strings")
		}

		token, err := big.FromString(s)
		if err != nil {
			return err
		}
		dbt.Token = token
		return nil
	case int64:
		s, ok := v.(int64)
		if !ok {
			return fmt.Errorf("DbToken must be strings")
		}

		token := big.NewInt(s)
		dbt.Token = token
		return nil
	}
	panic("here")
}

func (dbt DbToken) Value() (driver.Value, error) {
	return dbt.Token.String(), nil
}

func (DbToken) GormDataType() string {
	return "bigint"
}
