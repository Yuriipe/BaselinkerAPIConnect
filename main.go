package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Invtr struct {
	Status      string
	Inventories []map[string]interface{}
}

var payload = []byte(`method=getInventories`)

func getJSON(payload []byte) []byte {
	var (
		Url     string = "https://api.baselinker.com/connector.php"
		Tok_val string = "4005311-4011334-B6DI5PO6AM7GZ1D80O21R8W7OFFOH41W147FM3KTNHBJ9ZDHFLX0NONB1OZPWLXG"
	)

	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}

	req.Header.Set("X-BLToken", Tok_val)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	response, err0 := client.Do(req)
	if err0 != nil {
		panic(err0)
	}

	defer response.Body.Close()

	ResponseBody, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var inven Invtr

	err4 := json.Unmarshal(ResponseBody, &inven)
	if err4 != nil {
		panic("Unable to unmarshall")
	}

	fmt.Println(inven)
	return ResponseBody
}

func createFile(arr []byte) {

	file, err := os.Create("JSON.txt")
	if err != nil {
		panic("Unable to create file")
	}

	defer file.Close()

	_, err2 := file.WriteString(string(arr))
	if err2 != nil {
		panic("Unable to write to file: JSON.txt")
	}
}

//func transformJSON(){}

//func pushJSONtoSQL() {}

func main() {
	createFile(getJSON(payload))
}
