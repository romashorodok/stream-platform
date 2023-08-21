package request

import "encoding/json"


func UnmarshalRequest[T any](data map[string]string) (*T, error) {
	jsonConverted, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	var resp T

	err = json.Unmarshal(jsonConverted, &resp)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}
