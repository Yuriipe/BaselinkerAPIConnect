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

var payload = []byte(`method=getInventoryProductsStock&parameters=%7B%22inventory_id%22%3A%2223251%22%7D`)

type baselinkerValue struct {
	ID    string
	Value int
}

type baselinkerProduct struct {
	ProductID int
	Stock     []baselinkerValue
	//Reservations []baselinkerValue
}

type N = map[string]interface{}

// getting JSON from BL
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
	//defining client to connect to BL
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	//recieving response and pushing it into response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	//defining variable to operate the response
	res := N{}

	//unmarshalling JSON into res
	if err := json.Unmarshal(responseBody, &res); err != nil {
		return nil, err
	}

	//assigning value to "products variable"
	products := []baselinkerProduct{}
	for k, v := range res["products"].(N) {
		product := baselinkerProduct{}
		if v, err := strconv.Atoi(k); err == nil {
			product.ProductID = v
		} else {
			return nil, err
		}

		product.Stock = toBaselinkerValue(v.(N), "stock")
		//product.Reservations = toBaselinkerValue(v.(N), "reservations")
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

func dataBaseQuery() {
	dbq, err := sql.Open("mysql", "srv56775_APIgolang:APIgolang123!@tcp(h27.seohost.pl:3306)/srv56775_APIgolang")
	if err != nil {
		panic("Unable to connect to MySQL")
	}

	defer dbq.Close()

	err = dbq.Ping()
	if err != nil {
		fmt.Println("Error verifying connection with DB")
		panic(err.Error())
	}
	fmt.Println("Connection sucessful")
	defer dbq.Close()

	SQLQ := "INSERT INTO `test_table` (`ID`, `Stock`) VALUES (?, ?);"
	impSQL, err := dbq.Prepare(SQLQ)
	if err != nil {
		panic(err)
	}
	defer impSQL.Close()

	fmt.Println("Done")
}

func main() {
	products, err := getBaselinkerJSON(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getJSON: %v\n", err)
		os.Exit(1)
	}
	for _, product := range products {
		fmt.Printf("%+v\n", product)
	}
	//dataBaseQuery()
}

/*
1. Find a struct with unmarshalled JSON
2. Export struct
3. Define sql.Prepare query
4. test sql query
*/
