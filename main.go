package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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

type UpdateFilterPair struct {
	Update bson.D
	Filter bson.D
}

// payload for baselinkerConnect queries
func setPayload(args string) []byte {
	// set date X days before current, for getting orders
	days, err := strconv.Atoi(getEnv("GO_DAYSBEFORE"))
	if err != nil {
		panic("failed getting days env from payload.env")
	}
	date := time.Now().AddDate(0, 0, days).Unix()
	setParameter := map[string]interface{}{"date_confirmed_from": date, "get_unconfirmed_orders": false}
	jsonData, err := json.Marshal(setParameter)
	if err != nil {
		panic("payload parameter marshal failed")
	}
	getOrdersParameters := string(jsonData)

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
		parametersVal = getOrdersParameters
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
func baselinkerConnect(url, token string, payload []byte) ([]byte, error) {
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

// returns stock values from BL
func getStock(body []byte) []interface{} {
	fmt.Println("Getting stock from response")
	var stock BLStockResponse
	err := json.Unmarshal(body, &stock)
	if err != nil {
		panic("unmarshal failed")
	}

	var toDB []interface{}
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

// returns price values from BL
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

// returns product amount from orders of required period from BL
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

// creates multiple docs in DB from given []interface{}
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

// updates db field value based on filter and update options
func (mdb *MongoDB) dbUpdate(uri, db, collection string, update, filter bson.D) {
	fmt.Println("Update start")

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
	result, err := coll.UpdateMany(context.TODO(), filter, update)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Price values succesfully updated\n Matched document: %v\n Modified documents: %v\n", result.MatchedCount, result.ModifiedCount)

}

// returns product id's with prices
func (mdb *MongoDB) getFromDB(uri, db, collection string) {
	fmt.Println("Update start")

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
	// Find documents
	cursor, err := coll.Find(context.Background(), bson.M{}, options.Find().SetProjection(bson.M{"_id": 1, "price": 1}))
	if err != nil {
		panic(err)
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			panic(err)
		}

		fmt.Println(result)
	}

}

// set db field values based on BL values
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
		_, err := coll.UpdateOne(context.TODO(), filter, updateDoc)
		if err != nil {
			return err
		}
	}
	fmt.Println("\nField update finished")

	return nil
}

// drops all documents in chosen collection
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

func artLogo() {
	art := `
	/***
	*    __________    _____    ___________________.____    .___ _______   ____  __.____________________ 
	*    \______   \  /  _  \  /   _____|_   _____/|    |   |   |\      \ |    |/ _|\_   _____|______   \
	*     |    |  _/ /  /_\  \ \_____  \ |    __)_ |    |   |   |/   |   \|      <   |    __)_ |       _/
	*     |    |   \/    |    \/        \|        \|    |___|   /    |    \    |  \  |        \|    |   \
	*     |______  /\____|__  /_______  /_______  /|_______ \___\____|__  /____|__ \/_______  /|____|_  /
	*            \/         \/        \/        \/         \/           \/        \/        \/        \/ 
	*                                                                                                    
	*        .__                                                                                         
	*      __|  |___                                                                                     
	*     /__    __/                                                                                     
	*        |__|                                                                                        
	*                                                                                                    
	*       _____   ________    _______    ________ ________    ________ __________                      
	*      /     \  \_____  \   \      \  /  _____/ \_____  \   \______ \\______   \                     
	*     /  \ /  \  /   |   \  /   |   \/   \  ___  /   |   \   |    |  \|    |  _/                     
	*    /    Y    \/    |    \/    |    \    \_\  \/    |    \  |        \    |   \                     
	*    \____|__  /\_______  /\____|__  /\______  /\_______  / /_______  /______  /                     
	*            \/         \/         \/        \/         \/          \/       \/                      
	*/
	`
	fmt.Println(art)
}

func showMenu() {
	fmt.Println("1 ---------------------Get prices from DB")
	fmt.Println("2 ------------Update product prices in DB")
	fmt.Println("3 ---Update orders from past 7 days in DB")
	fmt.Println("4 ---------------------Reset order values")
	fmt.Println("5 ----Run price update logic in DB and BL")
	fmt.Println("9 ------------Delete all products from DB")
	fmt.Println("0 -----------------Exit------------------")
}

func returnToMenu() {
	var returnChoice string
	fmt.Println("Do you want to return to the menu? y/n")
	fmt.Scan(&returnChoice)
	if returnChoice != "y" {
		fmt.Println("Exiting.....")
		os.Exit(0)
	}
}

func main() {
	if err := doMain(); err != nil {
		log.Fatalln(err)
	}
}

func doMain() error {
	artLogo()

	cred := Authorization{}

	err := gonfig.GetConf("config/auth.json", &cred)
	if err != nil {
		panic("unable to set creadentials from json")
	}

	if err := godotenv.Load("config/payloadCfg.env", "config/mongoCfg.env"); err != nil {
		panic("loading payloadCfg.env failed")
	}

	uri := getEnv("MONGODB_URI")
	db := getEnv("DATABASE_NAME")
	collection := getEnv("COLLECTION_NAME")
	mdb := MongoDB{}

	// creates products from BL in db and sets stock values
	stock, err := baselinkerConnect(cred.URL, cred.Token, setPayload("getInventoryProductsStock"))
	if err != nil {
		panic(err)
	}
	stocks := getStock(stock)
	mdb.dbCreateMulti(stocks, uri, db, collection)

	var choice int
	for {
		showMenu()
		fmt.Println("Type your choice number and press \"Enter\" to confirm")
		fmt.Scan(&choice)
		switch choice {
		case 1:
			//updates product prices in BL
			mdb.getFromDB(uri, db, collection)
		case 2:
			// updates prices fields in DB
			price, err := baselinkerConnect(cred.URL, cred.Token, setPayload("getInventoryProductsPrices"))
			if err != nil {
				panic(err)
			}
			prices := getPrice(price)
			mdb.dbUpdateFieldsFromBL(uri, db, collection, "price", prices)
		case 3:
			// updates orders fields in DB
			order, err := baselinkerConnect(cred.URL, cred.Token, setPayload("getOrders"))
			if err != nil {
				panic(err)
			}
			orders := getOrders(order)
			mdb.dbUpdateFieldsFromBL(uri, db, collection, "orders", orders)
		case 4:
			// set DB order value to 0, executes on demand and after DB price update
			ordToZeroUpdate := bson.D{{"$set", bson.M{"orders": 0}}}
			ordToZeroFilter := bson.D{{"orders", bson.M{"$gt": 0}}}
			mdb.dbUpdate(uri, db, collection, ordToZeroUpdate, ordToZeroFilter)
		case 5:
			// set DB price value according to price update logic
			pairs := []UpdateFilterPair{
				{
					Update: bson.D{{"$mul", bson.D{{"price", 1.1}}}},
					Filter: bson.D{{"stock", bson.M{"$gt": 50}}, {"orders", bson.M{"$gt": 0, "$lt": 10}}},
				},
				{
					Update: bson.D{{"$mul", bson.D{{"price", 1.2}}}},
					Filter: bson.D{{"stock", bson.M{"$gt": 50}}, {"orders", bson.M{"$gt": 10}}},
				},
				{
					Update: bson.D{{"$mul", bson.D{{"price", 0.9}}}},
					Filter: bson.D{{"stock", bson.M{"$gt": 50}}, {"orders", bson.M{"$eq": 0}}},
				},
			}
			for _, pair := range pairs {
				mdb.dbUpdate(uri, db, collection, pair.Update, pair.Filter)
			}
		case 9:
			mdb.dbDeleteAllProducts(uri, db, collection)
		case 0:
			var confirm string
			fmt.Println("Confirm exiting y/n")
			fmt.Scan(&confirm)
			if confirm != "y" {
				returnToMenu()
			} else {
				fmt.Println("Exiting.....")
				os.Exit(0)
			}
		default:
			fmt.Println("Please make the valid choice")
			returnToMenu()
		}
		if choice != 4 {
			returnToMenu()
		} else {
			break
		}
	}

	return nil
}
