package gossiper2_backend

import (
	"sync"
	"fmt"
	"github.com/sagap/Peerster/part1/messaging"
	"net"
	"strings"
)

type VectorClock struct {
	mu    sync.RWMutex
	store map[string]map[uint32]string
	muTxt sync.RWMutex
	messagesText []string
}

func (gossiper *Gossiper)NewVectorClock(commitC <-chan *messaging.GossipPacket) *VectorClock {
	vc := &VectorClock{
		store:			make(map[string]map[uint32]string),
		messagesText: 	[]string{},
	}
	go gossiper.readCommitChannel(commitC)
	return vc
}

func (kv *VectorClock) Lookup(origin string, ID uint32) (string, bool) {
	kv.mu.Lock()
	v, ok := kv.store[origin][ID]
	defer kv.mu.Unlock()
	return v, ok
}

func (gossiper *Gossiper) readCommitChannel(commit <-chan *messaging.GossipPacket) {
	for data := range commit{
		if data != nil {
			_, ok := gossiper.Vc.Lookup(data.Rumor.Origin,data.Rumor.ID)
			if gossiper.Vc.lastRecordToReturn2(data.Rumor.Origin)+1 == data.Rumor.ID || !ok{
				gossiper.Vc.Add(data.Rumor.Origin, data.Rumor.Text, data.Rumor.ID)
				if strings.Compare(data.Rumor.Text,"") != 0{
					go gossiper.Vc.storeMessage(data.Rumor.Origin+"->"+data.Rumor.Text)
				}
			}
		}
	}
}

func (kv *VectorClock) storeMessage(text string){
	kv.messagesText = append(kv.messagesText, text)
}

func (kv *VectorClock) Add(origin, text string, ID uint32) {
	kv.mu.Lock()
	child, ok := kv.store[origin]
	if !ok {
		child = map[uint32]string{}
		kv.store[origin] = child
	}
	kv.store[origin][ID] = text
	defer kv.mu.Unlock()
}

func (gossiper *Gossiper) CompareVectorClocks(msg messaging.MsgReceiver) bool{
	flag := true
	once := true
	peersExistent := make(map[string]bool)
	//peersExistent[gossiper.origin] = true
	for index := range msg.Msg.Status.Want {
		tempOrigin := msg.Msg.Status.Want[index].Identifier
		peersExistent[tempOrigin] = true
		tempVC, ok := gossiper.Vc.Lookup(tempOrigin, msg.Msg.Status.Want[index].NextID)
		if ok{
			flag = false
			tempPort := 0
			newPacket := messaging.GossipPacket{&messaging.RumorMessage{tempOrigin,msg.Msg.Status.Want[index].NextID,tempVC, &net.IP{}, &tempPort}, nil, nil}
			gossiper.sendMessages(newPacket, msg.UpdAddr)
			counter := msg.Msg.Status.Want[index].NextID
			for true{
				counter++
				tempText, ok1 := gossiper.Vc.Lookup(tempOrigin, counter)
				if ok1{
					newPacket := messaging.GossipPacket{&messaging.RumorMessage{tempOrigin,counter,tempText, &net.IP{}, &tempPort}, nil, nil}
					gossiper.sendMessages(newPacket, msg.UpdAddr)
				}else{
					break
				}
			}
		}else{
			if once{
				counter := msg.Msg.Status.Want[index].NextID - 1
				_, ok := gossiper.Vc.Lookup(tempOrigin, counter)
				if !ok {
					flag = false
					once = false
					gossiper.Vc.mu.RLock()
					newPacket := messaging.GossipPacket{nil, &messaging.StatusPacket{gossiper.CreateStatusPacket2()}, nil}
					gossiper.sendMessages(newPacket, msg.UpdAddr)
					gossiper.Vc.mu.RUnlock()
				}
			}
		}
	}
	gossiper.Vc.mu.RLock()
	for key, value := range gossiper.Vc.store{
		_, ok := peersExistent[key]
		if !ok{
			flag = false
			counter := 1
			tempPort := 0
			for true{
				tempText, ok := value[uint32(counter)]
				if ok{
					newPacket := messaging.GossipPacket{&messaging.RumorMessage{key,uint32(counter),tempText, &net.IP{}, &tempPort}, nil, nil}
					gossiper.sendMessages(newPacket, msg.UpdAddr)
					counter++
				}else{
					break
				}
			}
			//gossiper.sendMessages(newPacket, vc.UpdAddr)
		}
	}
 	defer gossiper.Vc.mu.RUnlock()
	if flag{
		fmt.Println("IN SYNC WITH",msg.UpdAddr)//, vc.Msg.Status)
	}
	return flag
}

func(gossiper *Gossiper) CreateStatusPacket2() []messaging.PeerStatus{
	peerStatus := []messaging.PeerStatus{}
	var temp messaging.PeerStatus
	mapping := make(map[string]messaging.PeerStatus)
	gossiper.Vc.mu.RLock()
	tempStore := gossiper.Vc.store
	defer gossiper.Vc.mu.RUnlock()
	for origin, _ := range tempStore{
		_, ok := mapping[origin]
		if !ok {
			temp = messaging.PeerStatus{origin,gossiper.Vc.lastRecordToReturn2(origin)+1}
			mapping[origin] = temp
		}
	}
	for _, temp := range mapping{
		peerStatus = append(peerStatus, temp)
	}
	return peerStatus
}

func (kv *VectorClock) lastRecordToReturn2(key string) uint32{
	max := -1
	kv.mu.RLock()
	tempStore := kv.store
	for k, _ := range tempStore[key]{
		temp := int(k)
		if temp > max{
			max = temp
		}
	}
	defer kv.mu.RUnlock()
	return uint32(max)
}

func (kv *VectorClock) Size() int{
	kv.mu.RLock()
	size := len(kv.store)
	kv.mu.RUnlock()
	return size
}

func (kv * VectorClock) ReturnMessagesText() []string{
	return kv.messagesText
}