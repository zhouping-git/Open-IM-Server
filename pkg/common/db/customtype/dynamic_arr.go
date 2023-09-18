package customtype

import (
	"database/sql/driver"
	"encoding/json"
)

type DynamicArr []interface{}

func (a *DynamicArr) Value() (driver.Value, error) {
	d, err := json.Marshal(a)
	return string(d), err
}

func (a *DynamicArr) Scan(v interface{}) error {
	return json.Unmarshal(v.([]byte), a)
}
