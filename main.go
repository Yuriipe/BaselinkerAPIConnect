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
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/tkanos/gonfig"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Authorization struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type ProductStock struct {
	ProductID    int            `json:"product_id"`
	Reservations map[string]int `json:"reservations"`
	Stock        map[string]int `json:"stock"`
}

type BLStockResponse struct {
	Status   string                  `json:"status"`
	Products map[string]ProductStock `json:"products"`
}

type ProductPrice struct {
	ProductID int                `json:"product_id"`
	Prices    map[string]float64 `json:"prices"`
}

type BLPriceResponse struct {
	ProductPrice map[string]ProductPrice `json:"products"`
}

type OrderedProducts struct {
	OrdProductID string `json:"product_id"`
	OrdQuantity  int    `json:"quantity"`
}

type Order struct {
	OrderedProducts []OrderedProducts `json:"products"`
}

type BLOrders struct {
	Orders []Order `json:"orders"`
}

type MongoDB struct {
	cl *mongo.Client
}

type N []interface{}

// payload for getBaselinkerJSON queries
func setPayload(args string) []byte {
	var methodVal, parametersVal string
	switch args {
	case "getInventoryProductsStock":
		methodVal = getEnv("GIPS_METHOD")
		parametersVal = getEnv("GIPS_PARAMETERS")
	case "getInventoryProductsPrices":
		methodVal = getEnv("GIPP_METHOD")
		parametersVal = getEnv("GIPP_PARAMETERS")
	case "getOrders":
		methodVal = getEnv("GO_METHOD")
		parametersVal = getEnv("GO_PARAMETERS")
	case "updateProductsPrices":
		methodVal = getEnv("UPP_METHOD")
		parametersVal = getEnv("UPP_PARAMETERS")
	}

	payload := url.Values{}
	payload.Add("method", methodVal)
	payload.Add("parameters", parametersVal)

	return []byte(payload.Encode())
}

// getting JSON from BL
func getBaselinkerJSON(url, token string, payload []byte) ([]byte, error) {
	fmt.Println("Getting values from BaseLinker")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		panic("bl request failed")
	}

	req.Header.Set("X-BLToken", token)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
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

func getStock(body []byte) []interface{} {
	fmt.Println("Getting stock from response")
	var stock BLStockResponse
	err := json.Unmarshal(body, &stock)
	if err != nil {
		panic("unmarshal failed")
	}

	var toDB N
	for _, product := range stock.Products {
		reserve := 0
		stock := 0
		for _, resVal := range product.Reservations {
			reserve += resVal
		}
		for _, stVal := range product.Stock {
			stock += stVal
		}

		productMap := bson.M{"_id": product.ProductID, "stock": stock, "price": 0, "orders": 0}
		toDB = append(toDB, productMap)
	}
	return toDB
}

func getPrice(body []byte) []bson.M {
	fmt.Println("Getting prices from response")
	var prices BLPriceResponse
	err := json.Unmarshal(body, &prices)
	if err != nil {
		panic("unmarshal failed")
	}

	var toDB []bson.M
	for _, product := range prices.ProductPrice {
		var value float64
		for name, price := range product.Prices {
			if name == "22333" {
				value = price
			}
		}

		priceMap := bson.M{"_id": product.ProductID, "price": value}
		toDB = append(toDB, priceMap)
	}
	return toDB
}

func getOrders(body []byte) []bson.M {
	fmt.Println("Getting orders from response")
	var orders BLOrders
	err := json.Unmarshal(body, &orders)
	if err != nil {
		panic("unmarshal failed")
	}

	quantitySum := make(map[int]int)
	var orderMap primitive.M
	var toDB []bson.M
	for _, order := range orders.Orders {
		for _, product := range order.OrderedProducts {
			val, err := strconv.Atoi(product.OrdProductID)
			if err != nil {
				panic("conversion failed")
			}
			quantitySum[val] += product.OrdQuantity
		}
	}

	for k, v := range quantitySum {
		orderMap = bson.M{"_id": k, "orders": v}
		toDB = append(toDB, orderMap)
	}
	return toDB
}

// create single value in DB
func (mdb *MongoDB) dbCreateMulti(value []interface{}, uri, db, collection string) {
	fmt.Println("Inserting values into DB")

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

	fmt.Println("Insert completed")
}

func (mdb *MongoDB) dbRead(value []interface{}, value2 interface{}) {

}

func (mdb *MongoDB) dbUpdate(uri, db, collection string, update, filter bson.M) {
	fmt.Println("Starting update")

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	opts := options.Update().SetUpsert(true)

	mdb.cl = client
	coll := mdb.cl.Database(db).Collection(collection)
	coll.UpdateMany(context.TODO(), filter, update, opts)

	fmt.Println("Update completed")

}

func (mdb *MongoDB) dbUpdateFieldsFromBL(uri, database, collection, field string, productFieldMaps []bson.M) error {
	if len(productFieldMaps) == 0 {
		return nil
	}
	fmt.Printf("Started %s field update in DB", field)

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	// Connect to the database and collection
	db := client.Database(database)
	coll := db.Collection(collection)

	// Create a filter document
	filter := bson.M{}
	for _, productFieldMap := range productFieldMaps {
		filter = bson.M{"_id": productFieldMap["_id"]}
		fmt.Println(productFieldMap)
		updateDoc := bson.M{"$set": bson.M{field: productFieldMap[field]}}
		fmt.Println(updateDoc)
		_, err := coll.UpdateOne(context.TODO(), filter, updateDoc)
		if err != nil {
			return err
		}
	}
	fmt.Println("\nField update finished")

	return nil
}

func (mdb *MongoDB) dbDeleteAllProducts(uri, database, collection string) error {
	fmt.Println("Deleting values from DB")

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	// Connect to the database and collection
	db := client.Database(database)
	coll := db.Collection(collection)

	// Delete all products
	_, err = coll.DeleteMany(context.TODO(), bson.M{})
	if err != nil {
		return err
	}

	fmt.Println("Cleanup successfully completed")

	return nil
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("env variable %s is not set", key))
	}
	return value
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

	if err := godotenv.Load("config/payloadCfg.env"); err != nil {
		panic("loading payloadCfg.env failed")
	}

	if err := godotenv.Load("config/mongoCfg.env"); err != nil {
		panic("loading mongoCfg.env failed")
	}

	uri := getEnv("MONGODB_URI")
	db := getEnv("DATABASE_NAME")
	collection := getEnv("COLLECTION_NAME")
	mdb := MongoDB{}

	// stock, err := getBaselinkerJSON(cred.URL, cred.Token, setPayload("getInventoryProductsStock"))
	// if err != nil {
	// 	panic(err)
	// }

	// price, err := getBaselinkerJSON(cred.URL, cred.Token, setPayload("getInventoryProductsPrices"))
	// if err != nil {
	// 	panic(err)
	// }

	order, err := getBaselinkerJSON(cred.URL, cred.Token, setPayload("getOrders"))
	if err != nil {
		panic(err)
	}

	// stocks := getStock(stock)
	// prices := getPrice(price)
	orders := getOrders(order)

	// updates prices fields in DB
	// mdb.dbUpdateFieldsFromBL(uri, db, collection, "price", prices)

	// updates orders fields in DB
	mdb.dbUpdateFieldsFromBL(uri, db, collection, "orders", orders)

	// set DB order value to 0, executes on demand and after DB price update
	// ordToZeroUpdate := bson.M{"$set": bson.M{"orders": 0}}
	// ordToZeroFilter := bson.M{"orders": bson.M{"$exists": true}}
	// mdb.dbUpdate(uri, db, collection, ordToZeroUpdate, ordToZeroFilter)

	// mdb.dbCreateMulti(stocks, uri, db, collection)

	// mdb.dbDeleteAllProducts(uri, db, collection)

	return nil
}
