package main

import (
	"fmt"

	"main.go/curiousity/payload"
)

func main() {
	m := "getOrders"
	p := "par"
	fmt.Println(payload.Creator(m, p))
}
