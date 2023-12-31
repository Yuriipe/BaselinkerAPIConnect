package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tkanos/gonfig"
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

type Authorization struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// contains different baselinker APi queries, can be extended as needed
type Payload struct {
	GetInventories             []string `json:"getInventories"`
	GetInventoryProductsStock  []string `json:"getInventoryProductsStock"`
	GetInventoryProductsPrices []string `json:"getInventoryProductsPrices"`
}

type PriceChanger struct {
	StartProductAmount  int
	FinishProductAmount int
}

type N = map[string]interface{}

// getting JSON from BL
func getBaselinkerJSON(url string, token string, payload []byte) ([]baselinkerProduct, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-BLToken", token)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	fmt.Println(req)
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

func main() {
	if err := doMain(); err != nil {
		panic(err)
	}
}

func doMain() error {
	cred := Authorization{}

	err := gonfig.GetConf("config/auth.json", &cred)
	if err != nil {
		panic("unable to set creadentials from json")
	}

	products, err := getBaselinkerJSON(cred.URL, cred.Token, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getJSON: %v\n", err)
		os.Exit(1)
	}
	for _, product := range products {
		fmt.Println(product.ProductID, product.Stock)
	}

	file, err := os.Create("prod.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header row
	writer.Write([]string{"id", "store_id", "quantity", "store_id", "quantity", "store_id", "quantity", "store_id", "quantity", "store_id", "quantity", "store_id", "quantity"})

	// Write data rows
	for _, product := range products {
		// return "product.Stock" values in slice of string format
		stockStr := make([]string, len(product.Stock))
		for k, value := range product.Stock {
			stockStr[k] = fmt.Sprint(value.ID, ",", value.Value)
		}
		fmt.Println(stockStr)
		stockStrCnv := strings.Join(stockStr, ",")
		// creates a slice of strings to add to CSV file prod.csv
		row := []string{strconv.Itoa(product.ProductID), ",", stockStrCnv}
		rowCnv := strings.Join(row, "")
		row = strings.Split(rowCnv, ",")
		err = writer.Write(row)
		if err != nil {
			panic(err)
		}

	}
	return nil
}
