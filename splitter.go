package msync

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"hash"
	"io"

	"github.com/chmduquesne/rollinghash/adler32"
)

// BlockSplitter 块切割器
type BlockSplitter struct {
	r          io.Reader
	w          io.Writer
	index      int
	blockSize  int
	offset     int64
	adlerHash  hash.Hash32
	sha256Hash hash.Hash
}

// NewBlockSplitter 创建一个块读取流，可从io.Reader中读取块并计算哈希值
func NewBlockSplitter(r io.Reader, blockSize int) *BlockSplitter {
	br := BlockSplitter{r: bufio.NewReader(r), adlerHash: adler32.New(), sha256Hash: sha256.New(), blockSize: blockSize}
	br.w = io.MultiWriter(br.adlerHash, br.sha256Hash)
	return &br
}

// Next 切割下一个blockSize大小的块，并计算哈希，如果是最后一块，size可能小于blockSize
func (r *BlockSplitter) Next() (*Block, error) {
	n, err := io.CopyN(r.w, r.r, int64(r.blockSize))
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}
		if n == 0 {
			return nil, err
		}
	}
	b := Block{Index: r.index, Offset: r.offset, Length: n, AdlerSum: r.adlerHash.Sum32(), Sha256Sum: r.sha256Hash.Sum(nil)}
	r.index++
	r.offset += n
	r.reset()
	return &b, nil
}

func (r *BlockSplitter) reset() {
	r.adlerHash.Reset()
	r.sha256Hash.Reset()
}
