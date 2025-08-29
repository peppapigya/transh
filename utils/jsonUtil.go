package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// JsonToStruct 将json转成结构体
func JsonToStruct(jsonString string, t interface{}) {
	err := json.Unmarshal([]byte(jsonString), t)
	if err != nil {
		fmt.Printf("错误：json解析失败：%v\n", err)
		os.Exit(1)
	}
	return
}

// ParserToJson 将结构体数据转成json
func ParserToJson(t interface{}) string {
	marshal, err := json.Marshal(t)
	if err != nil {
		fmt.Printf("错误：json解析失败：%v\n", err)
		os.Exit(1)
	}
	return string(marshal)
}
