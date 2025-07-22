package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

type Node struct {
	Root  string `json:"root"`
	Left  *Node  `json:"left,omitempty"`
	Right *Node  `json:"right,omitempty"`
}

func build(data [][]byte) *Node {
	// step 1: build leaf node
	leaves := make([]*Node, len(data))
	for i, d := range data {
		h := sha256.Sum256(d)
		leaves[i] = &Node{Root: hex.EncodeToString(h[:])}
	}

	for len(leaves) > 1 {
		// step 2: if it constains odd data length, duplicate the last one
		if len(leaves)%2 == 1 {
			leaves = append(leaves, leaves[len(leaves)-1])
		}

		parents := make([]*Node, len(leaves)/2)
		for i := 0; i < len(leaves); i += 2 {
			left, right := leaves[i], leaves[i+1]
			// step 3: concat and hash
			b1, _ := hex.DecodeString(left.Root)
			b2, _ := hex.DecodeString(right.Root)
			sum := sha256.Sum256(append(b1, b2...))

			parents[i/2] = &Node{
				Root:  hex.EncodeToString(sum[:]),
				Left:  left,
				Right: right,
			}
		}
		leaves = parents
	}

	return leaves[0]
}

func BuildMarkle(data [][]byte) []byte {
	tree := build(data)

	out, err := json.MarshalIndent(tree, "", " ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	return out

}
