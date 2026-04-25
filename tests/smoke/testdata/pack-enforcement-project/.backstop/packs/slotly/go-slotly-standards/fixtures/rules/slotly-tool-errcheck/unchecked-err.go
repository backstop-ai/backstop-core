package main

import (
	"fmt"
	"os"
)

func main() {
	f, _ := os.Open("data.txt")
	defer f.Close()

	// Error return from Read is silently discarded
	f.Read(make([]byte, 10))
	fmt.Println("done")
}
