package gossiper2_backend

import (
	"github.com/sagap/Peerster/hw3/part2/messaging"
	"os"
	"log"
	"fmt"
	"crypto/sha256"
	"hash"
	"time"
	"github.com/sagap/Peerster/hw3/part2/metadata"
	"strings"
	"encoding/hex"
	"math/rand"
	"sync"
)

const FixedSizeLimit  = 8000          // each chunk 8KB
const MaxBudget		  = 32
const Threshold 	  = 2            // as soons as I find 2 files

type FileSharing struct{
	channelDataRequest  <-chan *messaging.DataRequest
	channelDataRequest2 <-chan *messaging.DataRequest
	channelSearchReq    <-chan *messaging.SearchRequest
	channelSearchRep    <-chan *messaging.SearchReply
	channelThresReached	chan *bool
	hashFunction  	    hash.Hash
	channelsToReceive   map[string]chan *messaging.DataReply           // channels that keep track of requests from specific node
	trackOfChunksPerReq map[string]int 								   // keep track of which chunk is sent (-1 is download is completed)
	metadata            map[string]metadata.Metadata				   // the filename is the key and metadata is the struct for info of the file
	handleDuplicates    map[string]time.Time
	storeReplies	    map[string]FileInfo				//	 filename -> FileInfo
	budgetNow			uint64
	mu1					sync.RWMutex
}

type FileInfo struct{
	nodeChunks map[string][]uint64						//   origin -> chunkMap
	metafile   []byte
}

func (gossiper *Gossiper)NewFileSharing(chanDataReq<- chan *messaging.DataRequest, chanDataReq2<- chan *messaging.DataRequest, chanSearchReq<- chan *messaging.SearchRequest, chanSearchReply<- chan *messaging.SearchReply) *FileSharing{
	fS := &FileSharing{
		channelDataRequest:   chanDataReq,
		channelDataRequest2:  chanDataReq2,
		channelSearchReq:	  chanSearchReq,
		channelSearchRep:	  chanSearchReply,
		channelThresReached:  make(chan *bool),
		hashFunction:		  sha256.New(),
		channelsToReceive:	  make(map[string]chan *messaging.DataReply),
		metadata:			  make(map[string]metadata.Metadata),
		trackOfChunksPerReq:  make(map[string]int),
		handleDuplicates:     make(map[string]time.Time),
		storeReplies:		  make(map[string]FileInfo),
	}
	gossiper.createDataRequest()
	gossiper.WaitingDataRequest()
	gossiper.acceptSearchReq()
	gossiper.WaitingForSearchReplies()
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

func (gossiper *Gossiper) WaitingForSearchReplies() {
	go func() {
		for reply := range gossiper.FS.channelSearchRep {
			fi := make(map[string][]uint64)
			tempFI := new(FileInfo)
			for _,v := range reply.Results{
				fi[reply.Origin] = v.ChunkMap
				tempFI.nodeChunks = fi
				val, ok := gossiper.FS.storeReplies[v.FileName]
				if !ok{
					//tempStore := make(map[string]FileInfo)
					//tempStore[v.FileName] = *tempFI
					gossiper.FS.storeReplies[v.FileName] = *tempFI
				}else{
					val.nodeChunks[reply.Origin] = tempFI.nodeChunks[reply.Origin]
				}
				fmt.Println("FOUND match",v.FileName,"at node",reply.Origin,"budget=",gossiper.FS.budgetNow,"metafile=",string(v.MetafileHash),"chunks=",v.ChunkMap)
				go gossiper.AskForMetafile(v.FileName, reply.Origin, v.MetafileHash)
			}
		}
	}()
}

func (gossiper *Gossiper) AskForMetafile(filename, dest string, hash []byte) {
	request := messaging.DataRequest{gossiper.origin,dest,10,filename,hash}
	msg := &messaging.GossipPacket{nil, nil, nil, &request, nil,nil,nil,nil}
	go gossiper.sendPrivateMessages(*msg, dest)
	gossiper.FS.mu1.Lock()
	gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]= make(chan *messaging.DataReply)
	gossiper.FS.mu1.Unlock()
	Reply := <-gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]
	fmt.Println("DOWNLOADING metafile of",request.FileName,"from node",request.Origin)
	vc := gossiper.FS.storeReplies[request.FileName]
	vc.metafile = Reply.Data
	gossiper.FS.storeReplies[request.FileName] = vc
	gossiper.FS.checkIfThreshold()
}

func (fs* FileSharing) checkIfThreshold(){
	counter := 0
	for _, v := range fs.storeReplies{
		chunks := len(v.metafile) / 32
		for _, chun := range v.nodeChunks{
			if len(chun) == chunks{
				counter +=1
			}
		}
	}
	if counter == Threshold{
		flag := true
		fs.channelThresReached <- &flag
	}
}

func (gossiper *Gossiper) acceptSearchReq(){
	go func(){
		for searchReq := range 	gossiper.FS.channelSearchReq{
			gossiper.searchForKeyWords(*searchReq)
			searchReq.Budget = searchReq.Budget - 1
			if searchReq.Budget > 0{
				gossiper.redistributeRequest(*searchReq)
			}
		}
	}()
}
func (gossiper *Gossiper) performSearch(req messaging.SearchRequest){
	if req.Budget == 0{
		budget := uint64(2)
		flag := true
		gossiper.FS.budgetNow = budget
		for budget <= 32  && flag{
			select {
				case <- time.After(1*time.Second):
					req.Budget = budget
					gossiper.redistributeRequest(req)
					budget = budget * 2
					gossiper.FS.budgetNow = budget
					break
				case <-gossiper.FS.channelThresReached:
					flag =false
					fmt.Println("SEARCH FINISHED")
					break
			}
		}
	}else{
		gossiper.redistributeRequest(req)
	}
}
func (gossiper *Gossiper) redistributeRequest(req messaging.SearchRequest){
	if int(req.Budget) >= len(gossiper.setpeers){
		diff := int(req.Budget) - len(gossiper.setpeers)
		var bud uint64
		if diff == 0{
			bud = 1
			req.Budget = bud
			message := &messaging.GossipPacket{nil,nil,nil,nil,nil,&req,nil,nil}
			for _, peer := range gossiper.setpeers{
				gossiper.sendMessages(*message, peer)
			}
		}else if diff / len(gossiper.setpeers) > 0 && diff % len(gossiper.setpeers) == 0{
			bud = uint64(uint64(diff) / uint64(len(gossiper.setpeers)))
			message := &messaging.GossipPacket{nil,nil,nil,nil,nil,&req,nil,nil}
			for _, peer := range gossiper.setpeers{
				gossiper.sendMessages(*message, peer)
			}
		}else if diff / len(gossiper.setpeers) > 0 && diff % len(gossiper.setpeers) > 0{
			for _, peer := range gossiper.setpeers{
				bud = uint64(uint64(diff) / uint64(len(gossiper.setpeers))) + uint64(uint64(diff) % uint64(len(gossiper.setpeers)))
				req.Budget = bud
				diff = diff - int(bud)
				message := &messaging.GossipPacket{nil,nil,nil,nil,nil,&req,nil,nil}
				gossiper.sendMessages(*message, peer)
			}
		}
	}else{
		diff := int(req.Budget)
		for range gossiper.setpeers {
			r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
			var temp int32
			temp = int32(len(gossiper.setpeers))
			num := r.Int31n(10) % temp
			randomPeer := gossiper.setpeers[num]
			fmt.Println("Sends to", randomPeer)
			bud := diff / len(gossiper.setpeers)
			req.Budget = uint64(bud)
			diff = diff - int(bud)
			message := &messaging.GossipPacket{nil,nil,nil,nil,nil,&req,nil,nil}
			go gossiper.sendMessages(*message, randomPeer)
		}
	}
}

func (gossiper *Gossiper) checkDuplicates(req messaging.SearchRequest) bool{
	arr := strings.Join(req.Keywords, ",")
	tim, ok := gossiper.FS.handleDuplicates[req.Origin+arr]
	if ! ok{
		gossiper.FS.handleDuplicates[req.Origin+arr] = time.Now()
		return true
	}else{
		if time.Duration(tim.Second() -time.Now().Second()) > time.Duration(500*time.Millisecond){
			gossiper.FS.handleDuplicates[req.Origin+arr] = time.Now()
			return true
		}else{    // else Duplicates
			return false
		}
	}
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
		metaFile, numOfChunks, hMap, chunkList := fs.buildMetafile(data[:count])
		metaHash := fs.createMetaHash(metaFile)
		tempMeta.FileSize = count
		tempMeta.MetaFile = metaFile
		tempMeta.MetaHash = metaHash
		tempMeta.FileData = data[:count]
		tempMeta.HashChunk = hMap
		tempMeta.Indexes = chunkList
		//fmt.Println("FILE:",tempMeta.FileData)
		tempMeta.NumOfChunks = numOfChunks
		fs.metadata[request.FileName] = tempMeta
}

func (gossiper *Gossiper)searchForKeyWords(request messaging.SearchRequest){
	var res []*messaging.SearchResult
	fmt.Println("SEarches for keywords",request.Origin)
	for _, keyWord := range request.Keywords{
		for k, v := range  gossiper.FS.metadata{
			if strings.Contains(k, keyWord){       // match
				result := &messaging.SearchResult{k,v.MetaHash,gossiper.FS.metadata[k].Indexes}
				res = append(res, result)
			}
		}
	}
	// send SearchReply
	fmt.Println("STELNEI........")
	searchReply := messaging.SearchReply{gossiper.origin,request.Origin,10,res}
	message := &messaging.GossipPacket{nil,nil,nil,nil,nil,nil,&searchReply,nil}
	go gossiper.sendPrivateMessages(*message, request.Origin)
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
			msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply,nil,nil,nil}
		}else{
			chunk, ok2 := val.HashChunk[string(hash)]
			if !ok2{
				fmt.Println("ERRORRR no chunk found")
				return
			}
			reply := messaging.DataReply{gossiper.origin, destination, 10, filename, hash, chunk}
			msg = &messaging.GossipPacket{nil, nil, nil, nil, &reply,nil,nil,nil}
		}
	}
	go gossiper.sendPrivateMessages(*msg, destination)
}

func (gossiper *Gossiper) startReceiving(request messaging.DataRequest) {                          // sending request
	msg := &messaging.GossipPacket{nil, nil, nil, &request, nil,nil,nil,nil}
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
		msg := &messaging.GossipPacket{nil, nil, nil, &request, nil,nil,nil,nil}
		gossiper.sendPrivateMessages(*msg, request.Destination)
		// waiting for reply
		Reply := <-gossiper.FS.channelsToReceive[request.Destination+","+request.FileName]
		fmt.Println("DOWNLOADING",Reply.FileName,"chunk",tempChunks,"from",Reply.Origin)
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
		gossiper.FS.checkDataHash(Reply, false, gossiper.FS.metadata[request.FileName].MetaFile[32*(tempChunks-1):32*(tempChunks)])
		tempChunks = tempChunks + 1
	}
	f, err := os.Create("./_Downloads/"+request.FileName)
	if err == nil{
		f.Write(gossiper.FS.metadata[request.FileName].FileData)
	}else {
		log.Fatal(err)
	}
	f.Close()
	fmt.Println("RECONSTRUCTED file",request.FileName)
}

func (fs *FileSharing) checkHashes(a, b []byte) bool {
	return strings.Compare(string(a),string(b)) == 0
}

func (fs *FileSharing) checkDataHash(reply *messaging.DataReply , isMetaFile bool, toCheck []byte) bool{
	if isMetaFile{
		dst := fs.metadata[reply.FileName].MetaHash
		fmt.Println(string(dst), string(reply.HashValue))
		if strings.Compare(string(dst), string(reply.HashValue)) == 0{
			return true
		}else{
			return false
		}
	}
	if strings.Compare(string(toCheck), string(reply.HashValue)) == 0{
		return true
	}else{
		return false
	}
}

func (fs *FileSharing)buildMetafile(fileData []byte) ([]byte, int, map[string][]byte, []uint64){
	var chunk []byte
	var NumOfChunks int
	hmap := make(map[string][]byte)
	if FixedSizeLimit > len(fileData){
		NumOfChunks = 1
	}else if (len(fileData)/FixedSizeLimit) != 0{
		NumOfChunks = (len(fileData)/FixedSizeLimit) + 1
	}else {
		NumOfChunks = len(fileData)/FixedSizeLimit
	}
	metaFile := make([]byte,32*NumOfChunks)
	fmt.Println("File Data: ",len(fileData),"File chunks: ",NumOfChunks, "Size: ",32*NumOfChunks)
	chunkList := []uint64{}
	counter := 0
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
		counter+=1
		chunkList = append(chunkList, uint64(counter))
		hmap[string(b)] = temp
	}
	if len(fileData) > 0 {
		temp := fileData
		fs.hashFunction.Write(fileData)
		b := fs.hashFunction.Sum(nil)
		for _,v := range b{
			metaFile[metaCounter] = v
			metaCounter++
		}
		counter += 1
		chunkList = append(chunkList, uint64(counter))
		hmap[string(b)] = temp
	}
	return metaFile, NumOfChunks, hmap, chunkList
}

func (fs *FileSharing) createMetaHash(metafile []byte) []byte{
	fs.hashFunction.Write(metafile)
	metaHash := fs.hashFunction.Sum(nil)
	dst := make([]byte, hex.EncodedLen(len(metaHash)))
	hex.Encode(dst, metaHash)
	fmt.Println("MetaHash :",string(dst))
	return dst
}