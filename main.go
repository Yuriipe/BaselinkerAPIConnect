package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

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
	response, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer response.Body.Close()

	ResponseBody, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("JSON.txt")
	if err != nil {
		panic("Unable to create file")
	}

	defer file.Close()

	_, err2 := file.WriteString(string(ResponseBody))
	if err2 != nil {
		panic("Unable to write to file: JSON.txt")
	}

	fmt.Println(string(ResponseBody))
	return ResponseBody
}

//func transformJSON(){}

//func pushJSONtoSQL() {}

func main() {
	getJSON(payload)
}
