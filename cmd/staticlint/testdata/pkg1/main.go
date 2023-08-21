package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello world")
	os.Exit(2) // want "call os.Exit in main function"
}
