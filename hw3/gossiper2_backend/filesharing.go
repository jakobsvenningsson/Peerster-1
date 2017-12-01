package gossiper2_backend

import (
	"github.com/sagap/Peerster/hw3/messaging"
	"os"
	"log"
	"fmt"
	"crypto/sha256"
	"hash"
	"time"
	"github.com/sagap/Peerster/hw3/metadata"
	"strings"
	"encoding/hex"
)

const FixedSizeLimit  = 8000          // each chunk 8KB

type FileSharing struct{
	channelDataRequest  <-chan *messaging.DataRequest
	channelDataRequest2 <-chan *messaging.DataRequest
	hashFunction  	    hash.Hash
	channelsToReceive   map[string]chan *messaging.DataReply           // channels that keep track of requests from specific node
	channelsToSend      map[string]chan *messaging.DataRequest         // channels that keep track of messages to send to specific nodes
	metadata            map[string]metadata.Metadata
}

func (gossiper *Gossiper)NewFileSharing(chanDataReq<- chan *messaging.DataRequest, chanDataReq2<- chan *messaging.DataRequest) *FileSharing{
	fS := &FileSharing{
		channelDataRequest:   chanDataReq,
		channelDataRequest2:  chanDataReq2,
		hashFunction:		  sha256.New(),
		channelsToReceive:	  make(map[string]chan *messaging.DataReply),
	    channelsToSend:	  	  make(map[string]chan *messaging.DataRequest),
		metadata:			  make(map[string]metadata.Metadata),
	}
	gossiper.createDataRequest()
	gossiper.WaitingDataRequest()
	return fS
}

func (gossiper *Gossiper) createDataRequest(){
	go func(){
		for request := range gossiper.FS.channelDataRequest {
			request.Origin = gossiper.origin
			request.HopLimit = 10
			fmt.Println("EDWWWWWWWWWWWWWW", request)
			go gossiper.startReceiving(*request)
		}
	}()

}

//   function that receives request to send data
func (gossiper *Gossiper) WaitingDataRequest(){
	go func(){
		for request := range gossiper.FS.channelDataRequest2{
			fmt.Println("REQUEST TO START SENDING...")
			str_key := request.Origin + "," + request.FileName
			channelTemp := make(chan *messaging.DataRequest)
			gossiper.FS.channelsToSend[str_key] = channelTemp
			_, ok := gossiper.FS.metadata[request.FileName]
			if !ok {
				var tempMeta metadata.Metadata
				file, err := os.Open(request.FileName) // For read access.
				if err != nil {
					log.Println(err)
				}
				data := make([]byte, 1000000)
				count, err := file.Read(data)
				if err != nil {
					log.Fatal(err)
				}
			//	fmt.Printf("read %d bytes: %q\n", count, data[:count])
				metaFile := gossiper.FS.buildMetafile(data[:count])
				metaHash := gossiper.FS.createMetahash(metaFile)
				fmt.Println("Metahash: ", metaHash)
				tempMeta.FileSize = count
				tempMeta.MetaFile = metaFile
				tempMeta.MetaHash = metaHash
				gossiper.FS.metadata[request.FileName] = tempMeta
			}
			if gossiper.FS.checkHashes(gossiper.FS.metadata[request.FileName].MetaHash, request.HashValue){
				fmt.Println("Starting Sending...")
			}
		}
	}()
}

func (gossiper *Gossiper) startReceiving(request messaging.DataRequest) {
	msg := &messaging.GossipPacket{nil, nil, nil, &request, nil}
	fmt.Println("EDWWWWWWWWWWWWWWWWWWWWWWWWWW")
	for {
		gossiper.sendPrivateMessages(*msg, request.Destination)
		gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]= make(chan *messaging.DataReply)
		fmt.Println("EEEeee",request.Destination+","+request.FileName)
		select {
			case Reply := <-gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]:
				fmt.Println(Reply)
				if gossiper.FS.checkDataHash(Reply){
					break         // break only after it receives the metafile and hashes match
				}else {
					continue
				}
			case <-time.After(5 * time.Second):
				fmt.Println("Timeout")
				//gossiper.sendPrivateMessages(*msg, request.Destination)
				continue
		}
	}
	go gossiper.sendReceiveChunks(request)
}

func (gossiper *Gossiper) sendReceiveChunks(request messaging.DataRequest) {


}

func (fs *FileSharing) checkHashes(a, b []byte) bool {

	if a == nil || b == nil {
		return false;
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (fs *FileSharing) checkDataHash(reply *messaging.DataReply) bool{
	fs.hashFunction.Write(reply.Data)
	b := fs.hashFunction.Sum(nil)
	if strings.Compare(hex.EncodeToString(b), string(reply.HashValue)) == 0{
		return true
	}else{
		return false
	}
}

func (fs *FileSharing)buildMetafile(fileData []byte) []byte{
	var chunk []byte
	metaFile := make([]byte,32*((len(fileData)/FixedSizeLimit)+1))
	fmt.Println("File: ",32*((len(fileData)/FixedSizeLimit)+1))
	metaCounter := 0
	for len(fileData) >= FixedSizeLimit {
		chunk, fileData = fileData[:FixedSizeLimit], fileData[FixedSizeLimit:]
		fs.hashFunction.Write(chunk)
		b := fs.hashFunction.Sum(nil)
		for _, v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
	}
	if len(fileData) > 0 {
		fs.hashFunction.Write(fileData)
		b := fs.hashFunction.Sum(nil)
		for _,v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
	}
	return metaFile
}

func (fs *FileSharing) createMetahash(metafile []byte) []byte{
	fs.hashFunction.Write(metafile)
	metaHash := fs.hashFunction.Sum(nil)
	dst := make([]byte, hex.EncodedLen(len(metaHash)))
	hex.Encode(dst, metaHash)
	return dst
}

func (fs *FileSharing) receiveDataReply(reply messaging.DataReply) {
	go func(){

	}()
}