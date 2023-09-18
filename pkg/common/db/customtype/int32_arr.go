package customtype

import (
	"database/sql/driver"
	"encoding/json"
)

type Int32Arr []int32

func (a Int32Arr) Value() (driver.Value, error) {
	//if a == nil {
	//	return nil, nil
	//}
	d, err := json.Marshal(a)
	return string(d), err
}

func (a *Int32Arr) Scan(v interface{}) error {
	return json.Unmarshal(v.([]byte), a)
}

//
//func (a *Int32Arr) MarshalBinary() (data []byte, err error) {
//	if a == nil {
//		return nil, nil
//	}
//	fmt.Println("MarshalBinary")
//	return json.Marshal(a)
//}
//
//func (a *Int32Arr) UnmarshalBinary(data []byte) error {
//	fmt.Println("UnmarshalBinary")
//	return json.Unmarshal(data, a)
//}
