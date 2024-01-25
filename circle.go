package main

import (
	"fmt"
	"math"
)

func main() {
    var area, radius float32
		// const pi = 3.14159 // use math package to get math.Pi
		fmt.Println("Input Radius in meters")
		fmt.Scanf("%f", &radius)
		area = math.Pi * radius * radius
		fmt.Printf(" radius of %g meters; area is %g square meters \n", radius, area)
}
