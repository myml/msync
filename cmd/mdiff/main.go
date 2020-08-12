package main

import (
	"errors"
	"io"
	"log"
	"os"

	"github.com/myml/msync"
)

const blockSize = 1024 * 1024

func main() {
	old := os.Args[1]
	new := os.Args[2]

	oldFile, err := os.Open(old)
	if err != nil {
		panic(err)
	}
	defer oldFile.Close()
	newFile, err := os.Open(new)
	if err != nil {
		panic(err)
	}
	defer newFile.Close()

	var blocks []*msync.Block
	splitter := msync.NewBlockSplitter(newFile, blockSize)
	for {
		b, err := splitter.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		blocks = append(blocks, b)
	}
	var existsBlock []*msync.FindBlock
	finder := msync.NewBlockFinder(oldFile, blocks, blockSize)
	for {
		b, err := finder.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		existsBlock = append(existsBlock, b)
	}
	log.Println("total block", len(blocks), "exists block", len(existsBlock))
}
