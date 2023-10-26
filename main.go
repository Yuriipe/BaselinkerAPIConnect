package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type Invtr struct {
	Status      string
	Inventories []map[string]interface{}
}

var payload = []byte(`method=getInventoryProductsStock&parameters=%7B%22inventory_id%22%3A%2223251%22%7D`)

// func payloadCrt(method string, parameters string)

type baselinkerValue struct {
	ID    string
	Value int
}

type baselinkerProduct struct {
	ProductID    int
	Stock        []baselinkerValue
	Reservations []baselinkerValue
}

type N = map[string]interface{}

func getBaselinkerJSON(payload []byte) ([]baselinkerProduct, error) {
	var (
		baselinkerUrl      string = "https://api.baselinker.com/connector.php"
		baselinkerUrlToken string = "4005311-4011334-B6DI5PO6AM7GZ1D80O21R8W7OFFOH41W147FM3KTNHBJ9ZDHFLX0NONB1OZPWLXG"
	)

	req, err := http.NewRequest(http.MethodPost, baselinkerUrl, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-BLToken", baselinkerUrlToken)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	res := N{}

	if err := json.Unmarshal(responseBody, &res); err != nil {
		return nil, err
	}
	products := []baselinkerProduct{}
	for k, v := range res["products"].(N) {
		product := baselinkerProduct{}
		if v, err := strconv.Atoi(k); err == nil {
			product.ProductID = v
		} else {
			return nil, err
		}

		product.Stock = toBaselinkerValue(v.(N), "stock")
		product.Reservations = toBaselinkerValue(v.(N), "reservations")
		products = append(products, product)
	}

	return products, nil
}

func toBaselinkerValue(node N, key string) []baselinkerValue {
	values := []baselinkerValue{}
	for k, val := range node[key].(N) {
		value := baselinkerValue{ID: k}
		value.Value = int(val.(float64))
		values = append(values, value)
	}
	return values
}

func crtFile(arr []byte) {

	file, err := os.Create("pro_dat.txt")
	if err != nil {
		panic("Unable to create file")
	}

	defer file.Close()

	if _, err := file.WriteString(string(arr)); err != nil {
		panic("Unable to write to file")
	}
}

func dataBaseQuery() {
	dbq, err := sql.Open("mysql", "srv56775_APIgolang:APIgolang123!@tcp(h27.seohost.pl:3306)/srv56775_APIgolang")
	if err != nil {
		panic("Unable to connect to MySQL")
	}

	defer dbq.Close()

	SQLQ := "LOAD DATA LOCAL INFILE '/home/yubo/Documents/Baselinker API/API_test/pro_dat.txt' into table test_table(jsondata);"
	impSQL, err3 := dbq.Query(SQLQ)
	if err3 != nil {
		panic(err3)
	}
	defer impSQL.Close()

	fmt.Println("Done")
}

//func pushJSONtoSQL() {}

func main() {
	products, err := getBaselinkerJSON(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getJSON: %v\n", err)
		os.Exit(1)
	}
	for _, product := range products {
		fmt.Println(product.ProductID, product.Stock[0])
	}

	// crtFile(resultJSON)
	// dataBaseQuery()
}
