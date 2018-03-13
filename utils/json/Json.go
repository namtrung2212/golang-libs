package Json

import (
	"bytes"
	"encoding/json"
	"reflect"
)

func JSONToObject(params string, obj interface{}) interface{} {

	objType := reflect.TypeOf(obj)
	req := reflect.New(objType).Elem().Interface()

	if err := json.Unmarshal([]byte(params), &req); err != nil {
		return nil
	}
	return req
}

func JSONFromObject(obj interface{}) (string, error) {

	res, err := json.Marshal(&obj)
	if err != nil {
		return "", nil
	}
	return string(res), nil
}

func PrettyJSON(data interface{}) (string, error) {

	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(true)

	// try to encode json
	if err := encoder.Encode(data); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
