package utils

import (
	"encoding/json"
	"log"
)

func ToJsonStr(obj interface{}) string {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		log.Fatalln(err)
	}
	return string(jsonBytes)
}
