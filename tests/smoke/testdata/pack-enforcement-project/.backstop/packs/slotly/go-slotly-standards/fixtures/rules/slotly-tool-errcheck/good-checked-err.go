package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.Open("data.txt")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer f.Close()

	_, err = f.Read(make([]byte, 10))
	if err != nil {
		fmt.Println("read error:", err)
	}
}
