package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONB Interface for JSONB Field of yourTableName Table
type DbJson string

// Value Marshal
func (a DbJson) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *DbJson) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

func (DbJson) GormDataType() string {
	return "jsonb"
}
