package msync

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/chmduquesne/rollinghash/adler32"
)

// FindBlock 记录查找到的本地块，包含远程块信息和本地偏移
type FindBlock struct {
	*Block
	FindOffset int64
}

// BlockFinder 块查找器，使用滚动哈希进行查找，使用强哈希做验证
type BlockFinder struct {
	r         *bufio.Reader
	blockSize int
	blocks    []*Block
	blockMap  map[uint32][]*Block
	rollhash  *adler32.Adler32
	blockBuff bytes.Buffer
	offset    int64
}

// NewBlockFinder 创建块查找器，从流中查找相同的块
func NewBlockFinder(r io.Reader, blocks []*Block, blockSize int) *BlockFinder {
	m := make(map[uint32][]*Block)
	for i := range blocks {
		m[blocks[i].AdlerSum] = append(m[blocks[i].AdlerSum], blocks[i])
	}
	return &BlockFinder{
		r:         bufio.NewReaderSize(r, blockSize),
		blockSize: blockSize,
		blocks:    blocks,
		blockMap:  m,
		rollhash:  adler32.New(),
	}
}

// Next 查找下一个相同的块
func (f *BlockFinder) Next() (*FindBlock, error) {
	f.rollhash.Reset()
	f.blockBuff.Reset()
	n, err := io.CopyN(io.MultiWriter(f.rollhash, &f.blockBuff), f.r, int64(f.blockSize))
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}
		if n == 0 {
			return nil, err
		}
	}
	f.offset += n
	for {
		digest := f.rollhash.Sum32()
		if blocks, ok := f.blockMap[digest]; ok {
			data := f.blockBuff.Bytes()
			sha256digest := sha256.Sum256(data)
			for i := range blocks {
				if bytes.Equal(blocks[i].Sha256Sum, sha256digest[:]) {
					return &FindBlock{FindOffset: f.offset - blocks[i].Length, Block: blocks[i]}, nil
				}
			}
		}
		b, err := f.r.ReadByte()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
			break
		}
		_, err = f.blockBuff.ReadByte()
		if err != nil {
			return nil, err
		}
		err = f.blockBuff.WriteByte(b)
		if err != nil {
			return nil, err
		}
		f.rollhash.Roll(b)
		f.offset++
	}
	return nil, io.EOF
}
