package metadata

type Metadata struct {
	FileSize    int
	MetaFile    []byte
	MetaHash    []byte
	FileData    []byte
	HashChunk   map[string][]byte
	NumOfChunks int
	Indexes		[]uint64
}

