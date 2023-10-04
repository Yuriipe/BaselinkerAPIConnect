package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	manipulateJSON()
	JSONtoStruct()
}

func databaseQuery() {
	fmt.Println("MySQL tutorial")

	db, err := sql.Open("mysql", "srv56775_APIgolang:APIgolang123!@tcp(h27.seohost.pl:3306)/srv56775_APIgolang")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	inser := "DROP TABLE `new_test_table`"
	//inser := "CREATE TABLE `srv56775_APIgolang`.`new_test_table` (`1` INT NOT NULL , `2` INT NOT NULL , `3` INT NOT NULL , `4` INT NOT NULL ) ENGINE = InnoDB; "
	//test := "ALTER TABLE `test_table` ADD `1` INT NOT NULL ; "
	insert, err := db.Query(inser)

	// if there is an error inserting, handle it
	if err != nil {
		panic(err.Error())
	}

	defer insert.Close()

	if err == nil {
		fmt.Println("Succesfully completed")
	}
}

func manipulateJSON() {
	readJSON()
	var mapingJSON map[string]interface{}
	err := json.Unmarshal(readJSON(), &mapingJSON)
	if err != nil {
		panic("Unable to unmarshal. Try more.")
	}

	resp := mapingJSON
	b, err4 := json.MarshalIndent(resp, "", " ")
	if err4 != nil {
		panic("Unable to marshalIndent")
	}

	file, err3 := os.Create("nice_JSON.txt")
	if err3 != nil {
		panic("Unable to create nice_JSON")
	}

	_, err2 := file.WriteString(string(b))
	if err2 != nil {
		panic("Unable to write to file")
	}

	defer file.Close()

	fmt.Println(string(b))

}

func readJSON() []byte {
	Body, err := os.ReadFile("/home/yubo/Documents/Baselinker API/API_test/pro_dat.txt")
	if err != nil {
		panic("Unable to read")
	}
	return Body
}

type getInventoryProductsStockBL struct {
	//Status   string
	Products struct {
		Num52576583 struct {
			ProductID int `json:"product_id"`
			Stock     struct {
				Bl27589 int `json:"bl_27589"`
				Bl52598 int `json:"bl_52598"`
				Bl55473 int `json:"bl_55473"`
			} `json:"stock"`
			Reservations struct {
				Bl27589 int `json:"bl_27589"`
				Bl52598 int `json:"bl_52598"`
				Bl55473 int `json:"bl_55473"`
			} `json:"reservations"`
		} `json:"52576583"`
	} `json:"products"`
}

func JSONtoStruct() {
	jsonInput, err := os.ReadFile("/home/yubo/Documents/Baselinker API/API_test/pro_dat.txt")
	if err != nil {
		panic("Unable to read nice_JSON")
	}

	fmt.Println(jsonInput)
	var transformedJson getInventoryProductsStockBL

	err2 := json.Unmarshal(jsonInput, &transformedJson)
	if err2 != nil {
		panic("Unable to unmarshal JSON")
	}
	fmt.Println(transformedJson)

}
