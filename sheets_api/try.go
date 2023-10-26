package main

type Car struct{
	topSpeed int
	FuelConsump map[string]int
	make string
	power *Pow	
}

type Pow struct{
	NM map[string]int
	HP map[string]int
}

func main() {
	c1 := Car{
		210, 
		FuelConsump[
			"key"
		], 
			"Dodge", 
			&Pow{}}
}

