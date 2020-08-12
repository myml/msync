package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/myml/msync"
)

var gen string
var sync string
var url string
var output string
var blockSize int

func init() {
	flag.StringVar(&gen, "gen", "", "generate .msync file")
	flag.StringVar(&sync, "sync", "", "sync file, need url and out param")
	flag.StringVar(&url, "url", "", "remote url")
	flag.StringVar(&output, "out", "", "output file")
	flag.IntVar(&blockSize, "b", 1024, "block size kb")
	flag.Parse()
	blockSize *= 1024
	if len(gen) == 0 {
		if len(sync) == 0 || len(url) == 0 || len(output) == 0 {
			flag.PrintDefaults()
			os.Exit(0)
		}
	}
}

func main() {
	if len(gen) > 0 {
		generate()
		return
	}
	syncFile()
}

func generate() error {
	in := os.Stdin
	if gen != "-" {
		f, err := os.Open(gen)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	out := os.Stdout
	if len(output) > 0 {
		f, err := os.Open(output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	// 生成块信息，并使用json编码
	encoder := json.NewEncoder(out)
	splitter := msync.NewBlockSplitter(in, blockSize)
	for {
		b, err := splitter.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		err = encoder.Encode(b)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func syncFile() {
	// 解析.msync文件
	decoder := json.NewDecoder(os.Stdin)
	var result []*msync.Block
	for {
		var b msync.Block
		err := decoder.Decode(&b)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		result = append(result, &b)
	}
	// 查找本地文件中已存在的块
	f, err := os.Open(sync)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	existsBlock := make(map[int]*msync.FindBlock)
	finder := msync.NewBlockFinder(f, result, blockSize)
	for {
		b, err := finder.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		existsBlock[b.Index] = b
	}

	log.Println("total block", len(result), "local block", len(existsBlock))

	// 创建合并用的临时文件
	out, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// 复制本地块
	downloader := msync.NewBlockReaderFromReadSeeker(f)
	for _, b := range existsBlock {
		_, err = out.Seek(b.Offset, 0)
		if err != nil {
			panic(err)
		}
		bb := *b.Block
		bb.Offset = b.FindOffset
		r, err := downloader.BlockReader(&bb)
		if err != nil {
			panic(err)
		}
		h := sha256.New()
		_, err = io.CopyN(io.MultiWriter(out, h), r, b.Length)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if !bytes.Equal(h.Sum(nil), b.Sha256Sum) {
			panic("block check")
		}
	}
	// 复制远程块
	downloader = msync.NewBlockReaderFromHTTP(http.DefaultClient, url)
	for i := range result {
		b := result[i]
		if _, exists := existsBlock[b.Index]; exists {
			continue
		}
		_, err = out.Seek(b.Offset, 0)
		if err != nil {
			panic(err)
		}
		r, err := downloader.BlockReader(b)
		if err != nil {
			panic(err)
		}
		h := sha256.New()
		_, err = io.CopyN(io.MultiWriter(out, h), r, b.Length)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if !bytes.Equal(h.Sum(nil), b.Sha256Sum) {
			panic("block check")
		}
	}
}
