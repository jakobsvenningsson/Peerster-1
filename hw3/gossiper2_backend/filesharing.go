package gossiper2_backend

import (
	"github.com/sagap/Peerster/hw3/messaging"
	//"strings"
	"os"
	"log"
	"fmt"
	"crypto/sha256"
	"hash"
 )

const FixedSizeLimit  = 80

type FileSharing struct{
	channelDataRequest <-chan *messaging.DataRequest
	hashFunction  	   hash.Hash
}

func (gossiper *Gossiper)NewFileSharing(chanDataReq<- chan *messaging.DataRequest) *FileSharing{
	fS := &FileSharing{
		channelDataRequest:   chanDataReq,
		hashFunction :		  sha256.New(),
	}
	return fS
}

func (gossiper *Gossiper) WaitingDataRequest(){
	go func(){
		for request := range gossiper.FS.channelDataRequest{
			fmt.Println("EDWWWWWWWWWWWWWWWW", request.FileName)
			//if strings.Compare(request.Origin,gossiper.origin) != 0{
			//	return
			//}else{
				file, err := os.Open(request.FileName) // For read access.
				if err != nil {
					log.Println(err)
				}
				data := make([]byte, 1000000)
				count, err := file.Read(data)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("read %d bytes: %q\n", count, data[:count])
				gossiper.FS.buildMetafile(data[:count])
			//}
		}
	}()
}

func (fs *FileSharing)buildMetafile(fileData []byte) []byte{
	var chunk []byte
	metaFile := make([]byte,32*(len(fileData)/FixedSizeLimit+1)+(len(fileData)/FixedSizeLimit))
	fmt.Println("File: ",32*(len(fileData)/FixedSizeLimit+1)+(len(fileData)/FixedSizeLimit))
	metaCounter := 0
	//chunks := make([][]byte, 0, len(fileData)/FixedSizeLimit+1)
	for len(fileData) >= FixedSizeLimit {
		chunk, fileData = fileData[:FixedSizeLimit], fileData[FixedSizeLimit:]
		fs.hashFunction.Write(chunk)
		b := fs.hashFunction.Sum(nil)
		//chunks = append(chunks, b)
		for _,v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
		metaFile[metaCounter] = ','      // bytecode 44 for comma
		metaCounter++
	}
	if len(fileData) > 0 {
		fs.hashFunction.Write(fileData)
		b := fs.hashFunction.Sum(nil)
		for _,v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
		//chunks = append(chunks, b)
	}
	//fmt.Println("T:   ",chunks)
	return metaFile
}