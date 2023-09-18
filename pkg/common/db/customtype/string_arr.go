package customtype

import (
	"database/sql/driver"
	"encoding/json"
)

type StringArr []string

func (a StringArr) Value() (driver.Value, error) {
	//if a == nil {
	//	return nil, nil
	//}
	d, err := json.Marshal(a)
	return string(d), err
}

func (a *StringArr) Scan(v interface{}) error {
	return json.Unmarshal(v.([]byte), a)
}

//
//func (a *StringArr) MarshalBinary() (data []byte, err error) {
//	if a == nil {
//		return nil, nil
//	}
//	fmt.Println("MarshalBinary")
//	return json.Marshal(a)
//}
//
//func (a *StringArr) UnmarshalBinary(data []byte) error {
//	fmt.Println("UnmarshalBinary")
//	return json.Unmarshal(data, a)
//}
