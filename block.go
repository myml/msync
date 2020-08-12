package msync

// Block 文件块
type Block struct {
	Index     int
	Offset    int64
	Length    int64
	AdlerSum  uint32
	Sha256Sum []byte
}
