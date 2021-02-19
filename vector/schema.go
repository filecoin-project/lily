package vector

import (
	"encoding/base64"
	"encoding/json"
)

type Options map[string]interface{}

// Base64EncodedBytes is a base64-encoded binary value.
type Base64EncodedBytes []byte

func (b Base64EncodedBytes) String() string {
	return base64.StdEncoding.EncodeToString(b)
}

// MarshalJSON implements json.Marshal for Base64EncodedBytes
func (b Base64EncodedBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.String())
}

// UnmarshalJSON implements json.Unmarshal for Base64EncodedBytes
func (b *Base64EncodedBytes) UnmarshalJSON(v []byte) error {
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return err
	}

	if len(s) == 0 {
		*b = nil
		return nil
	}

	bytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = bytes
	return nil
}

type Metadata struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Network     string `json:"network"`
	Date        int64  `json:"time"`
}

type Parameters struct {
	From          int64    `json:"from"`
	To            int64    `json:"to"`
	Tasks         []string `json:"tasks"`
	AddressFilter string   `json:"address-filter"`
}

type BuilderExpected struct {
	Models map[string][]interface{}
}

type RunnerExpected struct {
	Models map[string]json.RawMessage
}
