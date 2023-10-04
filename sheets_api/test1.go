package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	jsonFile, err := os.ReadFile("/home/yubo/Documents/Baselinker API/API_test/sheets_api/nice_JSON.txt")
	if err != nil {
		panic("Unable to read file")
	}

	var unmarshJson map[string]interface{}
	err2 := json.Unmarshal(jsonFile, &unmarshJson)
	if err2 != nil {
		panic("Problem unmarshaling")
	}

	//delete(unmarshJson, "status")
	//var unmarshRepaired map[string]interface{}

	fmt.Println(unmarshJson)

	marshJson, err3 := json.MarshalIndent(unmarshJson, "", " ")
	if err3 != nil {
		panic(err3)
	}

	newFile, err4 := os.Create("Test1.txt")
	if err4 != nil {
		panic("Bad creator")
	}

	_, err5 := newFile.WriteString(string(marshJson))
	if err5 != nil {
		panic("Write interrupted")
	}
}
