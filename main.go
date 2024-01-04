package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/tkanos/gonfig"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var payload = []byte(`method=getInventoryProductsStock&parameters=%7B%22inventory_id%22%3A%2223251%22%7D`)

type Authorization struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type ProductStock struct {
	ProductID    int            `json:"product_id"`
	Reservations map[string]int `json:"reservations"`
	Stock        map[string]int `json:"stock"`
}

type BLResponse struct {
	Status   string                  `json:"status"`
	Products map[string]ProductStock `json:"products"`
}

type BLQuery struct {
	pld *[]PayloadMethods
}

type Payload struct {
	Method     string `json:"method"`
	Parameters string `json:"parameters"`
}

// contains different baselinker APi queries, can be extended as needed
type PayloadMethods struct {
	GetInventoryProductsStock  []Payload `json:"getInventoryProductsStock"`
	GetInventoryProductsPrices []Payload `json:"getInventoryProductsPrices"`
	GetOrders                  []Payload `json:"getOrders"`
	UpdateProductsPrices       []Payload `json:"updateProductsPrices"`
}

type PriceChanger struct {
	StartProductAmount  int
	FinishProductAmount int
}

type MongoDB struct {
	cl *mongo.Client
}

func setPayload(Payload) ([]byte, error) {
	cfg := Payload{}
	err := gonfig.GetConf("config/cfg.json", &cfg)
	if err != nil {
		panic("unable to get method and parameters")
	}

	mathodVal := os.Getenv(cfg.Method)
	parametersVal := os.Getenv(cfg.Parameters)

	payload := url.Values{}
	method := payload.Add("method=", methodVal)

	return payload, nil
}

// getting JSON from BL
func getBaselinkerJSON(url string, token string, payload []byte) ([]byte, error) {
	fmt.Println("Getting stock and reserve values from BaseLinker")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		panic("bl request failed")
	}

	req.Header.Set("X-BLToken", token)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	fmt.Println(req)
	//defining client to connect to BL
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		panic("bl client.Do failed")
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("Values downloaded")
	return body, nil
}

func stockUpdate(body []byte) ([]interface{}, error) {
	fmt.Println("Preparing database insert")
	var result BLResponse
	err := json.Unmarshal(body, &result)
	if err != nil {
		panic("unmarshal failed")
	}
	toDB := []interface{}{}
	for _, product := range result.Products {
		reserve := 0
		stock := 0
		for _, resVal := range product.Reservations {
			reserve += resVal
		}
		for _, stVal := range product.Stock {
			stock += stVal
		}

		productMap := bson.M{"_id": product.ProductID, "stock": stock, "reserved": reserve, "stock2": 0, "reserved2": 0}
		toDB = append(toDB, productMap)
	}
	return toDB, nil
}

// used for cron based stock and price, check and update
func (bl *BLQuery) blQueryMenu(query string, method string, parameters []string) {
	switch query {
	case "getStock":
		bl.blGetStock(method, parameters)
	case "getProducts":
		bl.blGetPrices(method, parameters)
	case "setPrices":
		bl.blSetPrices(method, parameters)
	}
}

func (bl *BLQuery) blGetStock(mtd string, param []string) {}

func (bl *BLQuery) blGetPrices(mtd string, param []string) {}

func (bl *BLQuery) blSetPrices(mtd string, param []string) {}

// create single value in DB
func (mdb *MongoDB) dbCreateMulti(value []interface{}) {
	fmt.Println("Inserting values into DB")

	if err := godotenv.Load("cfg.env"); err != nil {
		panic("unable to read from cfg.env")
	}

	uri := os.Getenv("MONGODB_URI")
	db := os.Getenv("DATABASE_NAME")
	collection := os.Getenv("COLLECTION_NAME")

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	mdb.cl = client

	coll := mdb.cl.Database(db).Collection(collection)
	coll.InsertMany(context.TODO(), value)
}

func (mdb *MongoDB) dbRead(value []interface{}, value2 interface{}) {

}
func (mdb *MongoDB) dbUpdate() {
	fmt.Println("starting update")
	if err := godotenv.Load("cfg.env"); err != nil {
		panic("unable to read from cfg.env")
	}

	uri := os.Getenv("MONGODB_URI")
	db := os.Getenv("DATABASE_NAME")
	collection := os.Getenv("COLLECTION_NAME")

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	// update sequence for MongoDB, check manual for more
	update := bson.M{"$set": bson.M{"stock2": 100}}
	filter := bson.M{"stock2": bson.M{"$exists": true}}

	mdb.cl = client
	coll := mdb.cl.Database(db).Collection(collection)
	coll.UpdateMany(context.TODO(), filter, update)

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

	response, err := getBaselinkerJSON(cred.URL, cred.Token, payload)
	if err != nil {
		panic(err)
	}

	toDB, err := stockUpdate(response)
	if err != nil {
		panic("failed processing stock update")
	}

	mdb := MongoDB{}
	fmt.Println(toDB...)
	// mdb.dbCreateMulti(toDB)
	mdb.dbUpdate()

	return nil
}
