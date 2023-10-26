package main

type square struct {
	side int
}

type rectangle struct {
	length int
	width  int
}

func area(r rectangle) int {
	area := r.length * r.width
	return area
}
