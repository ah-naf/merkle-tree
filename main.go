package main

import (
	"fmt"

	"github.com/ah-naf/merkle-cli/merkle"
)

func main() {
	data := [][]byte{
		[]byte("A"), []byte("B"),
	}

	output := merkle.BuildMarkle(data)
	fmt.Println(output)
}
