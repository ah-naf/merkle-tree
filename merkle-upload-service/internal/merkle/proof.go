package merkle

import (
	"crypto/sha256"
	"encoding/hex"
)

type ProofStep struct {
	Hash     string `json:"hash"`
	Position string `json:"position"` // "left" or "right"
}

type MerkleProof struct {
	Leaf  string      `json:"leaf"`
	Proof []ProofStep `json:"proof"`
	Root  string      `json:"root"`
}

func GenerateProof(root *Node, target string) (*MerkleProof, bool) {
	var steps []ProofStep

	var dfs func(n *Node) bool
	dfs = func(n *Node) bool {
		// Leaf node
		if n.Left == nil && n.Right == nil {
			return n.Root == target
		}

		// Search left subtree
		if n.Left != nil && dfs(n.Left) {
			// record sibling (right)
			steps = append(steps, ProofStep{Hash: n.Right.Root, Position: "right"})
			return true
		}
		// Search right subtree
		if n.Right != nil && dfs(n.Right) {
			// record sibling (left)
			steps = append(steps, ProofStep{Hash: n.Left.Root, Position: "left"})
			return true
		}
		return false
	}

	found := dfs(root)
	if !found {
		return nil, false
	}

	return &MerkleProof{
		Leaf:  target,
		Proof: steps,
		Root:  root.Root,
	}, true
}

func VerifyProof(proof MerkleProof) bool {
	current, err := hex.DecodeString(proof.Leaf)
	if err != nil {
		return false
	}

	for _, step := range proof.Proof {
		sibling, err := hex.DecodeString(step.Hash)
		if err != nil {
			return false
		}
		var combined []byte
		if step.Position == "left" {
			combined = append(sibling, current...)
		} else {
			combined = append(current, sibling...)
		}
		sum := sha256.Sum256(combined)
		current = sum[:]
	}

	return hex.EncodeToString(current) == proof.Root
}
