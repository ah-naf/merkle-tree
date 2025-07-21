package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ah-naf/merkle-cli/internals/merkle"
)

func CollectData(inFile, dirPath, delim string) ([][]byte, error) {
	if inFile != "" {
		raw, err := os.ReadFile(inFile)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", inFile, err)
		}
		parts := strings.Split(string(raw), delim)
		var data [][]byte
		for _, p := range parts {
			if p != "" {
				data = append(data, []byte(p))
			}
		}
		return data, nil
	}

	var files []string
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", dirPath, err)
	}
	sort.Strings(files)

	var data [][]byte
	for _, p := range files {
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", p, err)
		}
		data = append(data, raw)
	}
	return data, nil
}

func LoadTree(path string) (*merkle.Node, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tree file %s: %w", path, err)
	}
	var node merkle.Node
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, fmt.Errorf("parse tree JSON: %w", err)
	}
	return &node, nil
}

func FileHash(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read target file %s: %w", path, err)
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}
