package gossiper2_backend

import (
	"github.com/sagap/Peerster/hw3/part1/messaging"
	"os"
	"log"
	"fmt"
	"crypto/sha256"
	"hash"
	"time"
	"github.com/sagap/Peerster/hw3/part1/metadata"
	"strings"
	"encoding/hex"
)

const FixedSizeLimit  = 8000          // each chunk 8KB

type FileSharing struct{
	channelDataRequest  <-chan *messaging.DataRequest
	channelDataRequest2 <-chan *messaging.DataRequest
	hashFunction  	    hash.Hash
	channelsToReceive   map[string]chan *messaging.DataReply           // channels that keep track of requests from specific node
	trackOfChunksPerReq map[string]int 								   // keep track of which chunk is sent (-1 is download is completed)
	metadata            map[string]metadata.Metadata				   // the filename is the key and metadata is the struct for info of the file
}

func (gossiper *Gossiper)NewFileSharing(chanDataReq<- chan *messaging.DataRequest, chanDataReq2<- chan *messaging.DataRequest) *FileSharing{
	fS := &FileSharing{
		channelDataRequest:   chanDataReq,
		channelDataRequest2:  chanDataReq2,
		hashFunction:		  sha256.New(),
		channelsToReceive:	  make(map[string]chan *messaging.DataReply),
		metadata:			  make(map[string]metadata.Metadata),
		trackOfChunksPerReq:  make(map[string]int),
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
			var meta metadata.Metadata
			meta.MetaHash = request.HashValue
			gossiper.FS.metadata[request.FileName] = meta
			if len(request.HashValue) == 0{
				gossiper.FS.shareFile(*request)
			}else{
				fmt.Println("EDWWWWWWWWWWWW")
				go gossiper.startReceiving(*request)
			}
		}
	}()
}

//   function that receives request to send data
func (gossiper *Gossiper) WaitingDataRequest(){
	go func(){
		for request := range gossiper.FS.channelDataRequest2{
			if gossiper.FS.checkHashes(gossiper.FS.metadata[request.FileName].MetaHash, request.HashValue) { // maybe removed
			}
			gossiper.sendAfterRequest(request.Origin, request.FileName, request.HashValue)

		}
	}()
}

func (fs *FileSharing) shareFile(request messaging.DataRequest){
		var tempMeta metadata.Metadata
		file, err := os.Open(request.FileName) // For read access.
		if err != nil {
		log.Println(err)
		}
		data := make([]byte, 3000000)
		count, err := file.Read(data)
		if err != nil {
		log.Fatal(err)
		}
		//fmt.Printf("read %d bytes: %q\n", count, data[:count])
		metaFile, numOfChunks, hMap := fs.buildMetafile(data[:count])
		metaHash := fs.createMetaHash(metaFile)
		tempMeta.FileSize = count
		tempMeta.MetaFile = metaFile
		tempMeta.MetaHash = metaHash
		tempMeta.FileData = data[:count]
		tempMeta.HashChunk = hMap
		//fmt.Println("FILE:",tempMeta.FileData)
		tempMeta.NumOfChunks = numOfChunks
		fs.metadata[request.FileName] = tempMeta
}

// this function sends DataReply objects after a DataRequest is received.
func (gossiper *Gossiper) sendAfterRequest(destination, filename string, hash []byte) {                   // create DataReply after requested has been received
	/*strKey := destination+filename
	_, ok := gossiper.FS.trackOfChunksPerReq[strKey]
	var msg *messaging.GossipPacket
	if !ok {
		gossiper.FS.trackOfChunksPerReq[strKey] = 1
		reply := messaging.DataReply{gossiper.origin, destination, 10, filename, gossiper.FS.metadata[filename].MetaHash, gossiper.FS.metadata[filename].MetaFile}
		msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply}
	}else{
		if   gossiper.FS.trackOfChunksPerReq[strKey] > gossiper.FS.metadata[filename].NumOfChunks {// completed
			//TODO Do nothing here             gossiper.FS.trackOfChunksPerReq[strKey] = -1
			fmt.Println("EDWWWWWWw3333333333333333333333333")
		}else{
			k := gossiper.FS.trackOfChunksPerReq[strKey]
			if k != gossiper.FS.metadata[filename].NumOfChunks{
				fmt.Println("EDWWWWWWw2222222222222")
				reply := messaging.DataReply{gossiper.origin, destination, 10, filename, gossiper.FS.metadata[filename].MetaFile[32*(k-1): 32*k], gossiper.FS.metadata[filename].FileData[FixedSizeLimit*(k-1): FixedSizeLimit*k]}
				msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply}
				k = k +1
				gossiper.FS.trackOfChunksPerReq[strKey] = k
			}else{
				reply := messaging.DataReply{gossiper.origin, destination, 10, filename, gossiper.FS.metadata[filename].MetaFile[32*(k-1):], gossiper.FS.metadata[filename].FileData[FixedSizeLimit*(k-1):]}
				fmt.Println("EDWWWWWWw11111")
				msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply}
				k = k +1
				gossiper.FS.trackOfChunksPerReq[strKey] = k
			}
		}
	}*/
	var msg *messaging.GossipPacket
	val, ok := gossiper.FS.metadata[filename]
    if !ok {
		fmt.Println("ERRORRR no file found")
		return
	}else{
		if strings.Compare(string(hash), string(val.MetaHash)) == 0{
			reply := messaging.DataReply{gossiper.origin, destination, 10, filename, hash, val.MetaFile}
			msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply}
		}else{
			chunk, ok2 := val.HashChunk[string(hash)]
			if !ok2{
				fmt.Println("ERRORRR no chunk found")
				return
			}
			reply := messaging.DataReply{gossiper.origin, destination, 10, filename, hash, chunk}
			msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply}
		}
	}
	go gossiper.sendPrivateMessages(*msg, destination)
}

func (gossiper *Gossiper) startReceiving(request messaging.DataRequest) {                          // sending request
	msg := &messaging.GossipPacket{nil, nil, nil, &request, nil}
	for {
		gossiper.sendPrivateMessages(*msg, request.Destination)
		gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]= make(chan *messaging.DataReply)
		flag := false
		select {
			case Reply := <-gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]:
				if gossiper.FS.checkDataHash(Reply, true,[]byte{}){
					fmt.Println("DOWNLOADING metafile of",request.FileName,"from",request.Origin)
					flag = true
					tempMeta := gossiper.FS.metadata[request.FileName]
					tempMeta.MetaFile = Reply.Data
					tempMeta.NumOfChunks = len(tempMeta.MetaFile) /32
					gossiper.FS.metadata[request.FileName] = tempMeta
					break         // break only after it receives the metafile and hashes match
				}else {
					continue
				}
			case <-time.After(5 * time.Second):
				fmt.Println("Timeout")
				//gossiper.sendPrivateMessages(*msg, request.Destination)
				continue
		}
		if flag{ break
		}
	}
	go gossiper.sendReceiveChunks(request)
}

// this is the function that is used to receive all the chunks
func (gossiper *Gossiper) sendReceiveChunks(request messaging.DataRequest) {
	tempChunks := 1
	for tempChunks <= gossiper.FS.metadata[request.FileName].NumOfChunks{
		request.HashValue = gossiper.FS.metadata[request.FileName].MetaFile[32*(tempChunks-1):32*(tempChunks)]
		msg := &messaging.GossipPacket{nil, nil, nil, &request, nil}
		gossiper.sendPrivateMessages(*msg, request.Destination)
		fmt.Println("RE",tempChunks, request)
		Reply := <-gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]
		fmt.Println("DOWNLOADING",Reply.FileName,"chunk",tempChunks,"from",Reply.Origin)
		///fmt.Println("EDWWWWWWWWWWWW", Reply.Data,"HASH: ",Reply.HashValue)
		tempMeta := gossiper.FS.metadata[request.FileName]
		metaCounter := len(tempMeta.FileData)
		if metaCounter == 0{
			tempMeta.FileData = make([]byte,len(Reply.Data))
			for _, v := range Reply.Data{
				tempMeta.FileData[metaCounter] = v
				metaCounter++
			}
		}else{
			for _, v := range Reply.Data{
				tempMeta.FileData = append(tempMeta.FileData, v)
			}
		}
		gossiper.FS.metadata[request.FileName] = tempMeta
		//fmt.Println("FTIAXNETAI:",gossiper.FS.metadata)
		gossiper.FS.checkDataHash(Reply, false, gossiper.FS.metadata[request.FileName].MetaFile[32*(tempChunks-1):32*(tempChunks)])
		tempChunks = tempChunks + 1
	}
	f, err := os.Create("./_Downloads/"+request.FileName)
	if err == nil{
		//fmt.Println("GRAFEIIIIIIIIIIIIIIIIIii")
		f.Write(gossiper.FS.metadata[request.FileName].FileData)
	}else {
		log.Fatal(err)
	}
	f.Close()
	fmt.Println("RECONSTRUCTED file",request.FileName)
}

func (fs *FileSharing) checkHashes(a, b []byte) bool {

	//if a == nil || b == nil {
	//	return false;
	//}
	//if len(a) != len(b) {
	//	return false
	//}
	//for i := range a {
	//	if a[i] != b[i] {
	//		return false
	//	}
	//}
	return strings.Compare(string(a),string(b)) == 0
}

func (fs *FileSharing) checkDataHash(reply *messaging.DataReply , isMetaFile bool, toCheck []byte) bool{
	if isMetaFile{
		dst := fs.metadata[reply.FileName].MetaHash
		fmt.Println(string(dst), string(reply.HashValue))
		if strings.Compare(string(dst), string(reply.HashValue)) == 0{
			//fmt.Println("OKKKKKKKKKKKKKKKKKKKKKKK")
			return true
		}else{
			//fmt.Println("NNNNNNNNNNNNNNNNNNNNNNNNOKKKKKKKKKKKKKKKKKKKKKKK")
			return false
		}
	}
	//fmt.Println("CHECKS: ",toCheck," --- ",reply.HashValue)
	if strings.Compare(string(toCheck), string(reply.HashValue)) == 0{
		return true
	}else{
		return false
	}
}

func (fs *FileSharing)buildMetafile(fileData []byte) ([]byte, int, map[string][]byte){
	var chunk []byte
	var NumOfChunks int
	hmap := make(map[string][]byte)
	//fmt.Println("FILEDATA: ",fileData)
	if FixedSizeLimit > len(fileData){
		NumOfChunks = 1
	}else if (len(fileData)/FixedSizeLimit) != 0{
		NumOfChunks = (len(fileData)/FixedSizeLimit) + 1
	}else {
		NumOfChunks = len(fileData)/FixedSizeLimit
	}
	metaFile := make([]byte,32*NumOfChunks)
	fmt.Println("File Data: ",len(fileData),"File chunks: ",NumOfChunks, "Size: ",32*NumOfChunks)
	metaCounter := 0
	for len(fileData) >= FixedSizeLimit {
		chunk, fileData = fileData[:FixedSizeLimit], fileData[FixedSizeLimit:]
		temp := chunk
		fs.hashFunction.Write(chunk)
		b := fs.hashFunction.Sum(nil)
		for _, v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
		hmap[string(b)] = temp
		//fmt.Println("chunk",temp,"string",string(b), hmap)
	}
	if len(fileData) > 0 {
		temp := fileData
		fs.hashFunction.Write(fileData)
		b := fs.hashFunction.Sum(nil)
		for _,v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
		hmap[string(b)] = temp
	}
	//fmt.Println("Chunk Data: ",hmap)
	return metaFile, NumOfChunks, hmap
}

func (fs *FileSharing) createMetaHash(metafile []byte) []byte{
	fs.hashFunction.Write(metafile)
	metaHash := fs.hashFunction.Sum(nil)
	dst := make([]byte, hex.EncodedLen(len(metaHash)))
	hex.Encode(dst, metaHash)
	fmt.Println("DST :",string(dst))
	return dst
}