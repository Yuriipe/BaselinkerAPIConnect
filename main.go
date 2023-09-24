package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	//set url
	url := "https://api.baselinker.com/connector.php"
	//add token
	tok_val := "4005311-4011334-B6DI5PO6AM7GZ1D80O21R8W7OFFOH41W147FM3KTNHBJ9ZDHFLX0NONB1OZPWLXG"
	//set method
	payload := []byte(`method=getInventories`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}

	req.Header.Set("X-BLToken", tok_val)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(responseBody))

}
