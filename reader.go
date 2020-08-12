package msync

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// BlockReader 块下载器的接口
type BlockReader interface {
	BlockReader(*Block) (io.ReadCloser, error)
}

// NewBlockReaderFromReadSeeker 从seeker创建块读取器
func NewBlockReaderFromReadSeeker(r io.ReadSeeker) BlockReader {
	return &blockReaderReadSeeker{r: r}
}

type blockReaderReadSeeker struct {
	r io.ReadSeeker
}

func (r *blockReaderReadSeeker) BlockReader(b *Block) (io.ReadCloser, error) {
	_, err := r.r.Seek(b.Offset, 0)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(io.LimitReader(r.r, b.Length)), nil
}

// NewBlockReaderFromHTTP 从http/https创建快读取器
func NewBlockReaderFromHTTP(client *http.Client, url string) BlockReader {
	return &blockReaderHTTP{client: client, url: url}
}

type blockReaderHTTP struct {
	url    string
	client *http.Client
}

func (r *blockReaderHTTP) BlockReader(b *Block) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", b.Offset, b.Offset+b.Length))
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf(resp.Status)
	}
	return resp.Body, nil
}
