package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {

	var url = "https://go.dev"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Unable to read the response")
	}

	file, err := os.Create("url_response.txt")
	if err != nil {
		fmt.Println("Unable to create a file")
	}

	defer file.Close()

	_, err2 := file.WriteString(string(body))
	if err2 != nil {
		fmt.Println("Unable to write")
	}

}
