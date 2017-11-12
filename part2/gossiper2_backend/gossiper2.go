package gossiper2_backend

import (
	"net"
	"log"
	"fmt"
	"strings"
	"strconv"
	"github.com/dedis/protobuf"
	"math/rand"
	"time"
	"sync"
	"github.com/sagap/Peerster/part2/messaging"
	"github.com/sagap/Peerster/part2/routing"
)

const localAddress = "127.0.0.1"

type Gossiper struct {
	address 	 *net.UDPAddr
	conn		 *net.UDPConn
	origin		 string							// origin
	msgChn1  	 chan messaging.GossipPacket   // channel to send messages from clients to other peers
	msgChn2		 chan messaging.MsgReceiver	// channel to send messages from peers to other peers
	msgChn3		 chan messaging.MsgReceiver   // channel for status messages
	msgChnACK	 chan messaging.MsgReceiver // channel for ACK
	msgChnRumor  chan messaging.RumorMessage   // channel for chat rumors and route rumors
	msgChnPriv	 chan messaging.PrivateMessage
	setpeers	 []string//map[int]string // known peers
	msgID		 uint32						// sequence of messages originated from this node
	vectorClock  map[string]string
	ticker       *time.Ticker
	waitForACK	 map[string]bool
	mu    		 sync.RWMutex				// mutex for vector clock
	mu2			 sync.RWMutex              // mutex for list of nodes that wait for ACK
	mupriv		 sync.RWMutex				// mutex for private messages
	Routerr		 routing.Router           // Router that implements Routing protocol of exrs 2
	ip			 *net.IP
	port		 *int
	peerNames	 []string
	GuiPort		 int
  	timeofMessage map[string]time.Time
	privateList  map[time.Time]messaging.PrivateMessage
	noForward	 bool
}

func NewGossiper(address, name string, peers string, rtimer, guiPort int, noforward bool) (*Gossiper){
	udpAddr, err1 := net.ResolveUDPAddr("udp4", address)
	udpConn, err2 := net.ListenUDP("udp4", udpAddr)
	if err1 != nil{
		log.Fatal("Error1: ",err1)
	}
	if err2 != nil{
		log.Fatal("Error2: ",err2)
	}

	var peersList []string
	if strings.Compare(peers,"") != 0{
		peersList = strings.Split(peers,"_")
	}

	gossiper := &Gossiper{
		address:	  udpAddr,
		conn:	 	  udpConn,
		msgID:		  0,
		origin:		  name,
		msgChn1:	  make(chan messaging.GossipPacket),
		msgChn2:	  make(chan messaging.MsgReceiver),
		msgChn3:	  make(chan messaging.MsgReceiver),
		msgChnACK:	  make(chan messaging.MsgReceiver),
		msgChnRumor:  make(chan messaging.RumorMessage),
		msgChnPriv:   make(chan messaging.PrivateMessage),
		setpeers:	  peersList,
		vectorClock:  make(map[string]string),
		ticker:		  time.NewTicker(10 * time.Second),
		waitForACK:   make(map[string]bool),
		peerNames:	  []string{},
		GuiPort:	  guiPort,
		timeofMessage:make(map[string]time.Time),
		privateList:  make(map[time.Time]messaging.PrivateMessage),
		noForward:	  noforward,
		}
	gossiper.Routerr = *routing.NewRouter(gossiper.msgChnRumor, rtimer)
	gossiper.ip = &(gossiper.address.IP)
	gossiper.port = &(gossiper.address.Port)
	go gossiper.listenToPeers()
	gossiper.waitForClientsToSend()         // function that handles the incoming messages from clients
	gossiper.waitForPeersToSend()			// function that handles the incoming messages from peers
	gossiper.waitForPeersToSendStatus()
	gossiper.acceptPointToPointMessage()
	go gossiper.AntiEntropy()
	if gossiper.GuiPort != 0{
		go gossiper.ServeHttpKVAPI()
	}
	go gossiper.announceMyself()
	return gossiper
}

func (g *Gossiper) listenToPeers(){
	for {
		packetBytes := make([]byte, 2048)
		n, udpAddr, err := g.conn.ReadFromUDP(packetBytes)
		go g.handleRequests(packetBytes, n, udpAddr.String(), true)
		if err != nil {
			log.Println("error during read: %s", err)
		}
		packetBytes = nil
	}
}

func (g *Gossiper) ServeClients(clientsPort int){
	udpAddress, err := net.ResolveUDPAddr("udp4", localAddress+":"+strconv.Itoa(clientsPort))
	listener, err := net.ListenUDP("udp", udpAddress)
	if err != nil{
		log.Fatal("Error3: ", err)
	}
	for {
		packetBytes := make([]byte, 2048)
		n, udpAddr, err := listener.ReadFromUDP(packetBytes)
		go g.handleRequests(packetBytes, n, udpAddr.String(),false)
		if err != nil {
			log.Println("error during read: %s", err)
		}
		packetBytes = nil
	}
}

func (gossiper *Gossiper)handleRequests(recv []byte, n int, udpAddr string, flag bool){
	t2 := messaging.GossipPacket{}
	err := protobuf.Decode(recv[:n], &t2)
	if err != nil {
		log.Fatal(err)
	}
	if gossiper.checkIfClient(t2){
		if !flag{                           // flag false for clients
			gossiper.msgChn1 <- t2          // channel for clients
		}
	}else{
		gossiper.appendPeersList(udpAddr)
		msgRecv := messaging.MsgReceiver{t2,udpAddr}      // keep the address of the relay peer into MsgReceiver channel
		if t2.Status != nil {
			gossiper.msgChn3 <- msgRecv
		}else if t2.Rumor != nil{
			if strings.Compare(t2.Rumor.Text,"") == 0 && strings.Compare(t2.Rumor.Origin, gossiper.origin) != 0{
				gossiper.msgChnRumor <- *msgRecv.Msg.Rumor
			}
			gossiper.msgChn2 <- msgRecv          // channel for Rumors
		}else if t2.Private != nil{
			gossiper.msgChnPriv <- *msgRecv.Msg.Private
		}
	}
}

func (gossiper *Gossiper)forwardMessage(msg messaging.GossipPacket, relayPeer string, randomPeer1 string){      // forward Message To random peer
	var randomPeer string
	if strings.Compare(randomPeer1," ") == 0{
		 r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		 var temp int32
		 temp = int32(len(gossiper.setpeers))
		 num := r.Int31n(10)%temp
		 randomPeer = gossiper.setpeers[num]
		if strings.Compare(randomPeer, relayPeer) == 0{
			if len(gossiper.setpeers) == 1{
				return
			}else{
				for{
					r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
					num = r.Int31n(10)%temp
					randomPeer = gossiper.setpeers[num]
					if strings.Compare(randomPeer, relayPeer) != 0{
						break
					}
				}
			 }
		 }
	 }else{
	 	randomPeer = randomPeer1
	 }
	gossiper.mu2.Lock()
	gossiper.waitForACK[randomPeer] = true
	gossiper.mu2.Unlock()
	fmt.Println("MONGERING with", randomPeer)
	go gossiper.sendMessages(msg, randomPeer)
	for{
		select{
			case vc := <-gossiper.msgChnACK:
				gossiper.mu2.Lock()
				gossiper.waitForACK[randomPeer] = false
				gossiper.mu2.Unlock()
				fmt.Print("STATUS from ",vc.UpdAddr);vc.Msg.Print()
				flagSync := gossiper.compareVectorClocks(vc)
				if flagSync{
					go gossiper.flipcoin(vc.Msg, vc.UpdAddr)
				}
				return
		    case <- time.After(1*time.Second): //<- t.C:
				gossiper.mu2.Lock()
		    	gossiper.waitForACK[randomPeer] = false
		    	gossiper.mu2.Unlock()
		//		fmt.Println("Timeout!!!!!!!!!!!!!!!", randomPeer)
				go gossiper.flipcoin(msg, relayPeer)
				return
		    }
	}
}

func (gossiper *Gossiper)waitForClientsToSend(){                         // input from channel1 creates Rumor Message to be sent to other peers
	go func() {
		for msg := range gossiper.msgChn1{
			gossiper.msgID++
			msg.Rumor.Origin = gossiper.origin
			msg.Rumor.ID = gossiper.msgID
			msg.Rumor.LastIP = gossiper.ip
			msg.Rumor.LastPort = gossiper.port
			fmt.Println("CLIENT",msg.Rumor.Text,msg.Rumor.Origin)
			gossiper.storeVectorClocks(msg)
			gossiper.forwardMessage(msg, " ", " ")
			gossiper.printKnownPeers()
		}
	}()
}

func (gossiper *Gossiper)waitForPeersToSend(){                        // input from channel2 msg received from peer
	go func() {
		for msg := range gossiper.msgChn2{
			if gossiper.checkMessageType(msg.Msg) {               // RumorMessage
				if strings.Compare(msg.Msg.Rumor.Origin, gossiper.origin) != 0{
					gossiper.msgChnRumor <- *msg.Msg.Rumor
				}
				if gossiper.ifMessageExists(msg.Msg.Rumor.Origin,msg.Msg.Rumor.ID) {
					continue
				}
				if strings.Compare(msg.Msg.Rumor.Text,"") != 0{
					fmt.Println("RUMOR origin",msg.Msg.Rumor.Origin,"from",msg.UpdAddr,"ID",msg.Msg.Rumor.ID,"contents",msg.Msg.Rumor.Text)
				}
				gossiper.storeVectorClocks(msg.Msg)
				if !gossiper.noForward || strings.Compare(msg.Msg.Rumor.Text,"") == 0 {
				toSend := messaging.GossipPacket{nil,&messaging.StatusPacket{gossiper.createStatusPacket()},nil}  			// ACK
				gossiper.sendMessages(toSend, msg.UpdAddr)
				msg.Msg.Rumor.LastIP = gossiper.ip
				msg.Msg.Rumor.LastPort = gossiper.port
				gossiper.forwardMessage(msg.Msg, msg.UpdAddr, " ")					// forward Rumor to other Peer
				}
			}
			gossiper.printKnownPeers()
		}
	}()
}

func (gossiper *Gossiper)waitForPeersToSendStatus(){                        // input from channel3 msg received from peer
	go func() {
		for msg := range gossiper.msgChn3{
			gossiper.mu2.RLock()
			_, found := gossiper.waitForACK[msg.UpdAddr]
			gossiper.mu2.RUnlock()
			if found{
				gossiper.msgChnACK <- msg
				continue
			}
			fmt.Print("STATUS from ",msg.UpdAddr);msg.Msg.Print()
			gossiper.compareVectorClocks(msg)
			//gossiper.msgChnACK<-msg // channel for statuspacket that goes to select
			}
			gossiper.printKnownPeers()
	}()
}

func (gossiper *Gossiper) sendRumors(msg messaging.RumorMessage, peers []string) {
	var message = &messaging.GossipPacket{&msg,nil,nil}
	/*packetBytes, err := protobuf.Encode(message)
	if err != nil {
		log.Fatal("Error4: ", err)
	}
	for _, peer := range peers {
		udpAddr, _ := net.ResolveUDPAddr("udp4", peer)
		fmt.Println("Send Rumor to ",udpAddr,",,,",message)
		gossiper.conn.WriteToUDP(packetBytes, udpAddr)
	}*/
	for _, peer := range peers {
		gossiper.forwardMessage(*message, "", peer)
	}
}

func(gossiper *Gossiper)acceptPointToPointMessage(){
	go func() {
		for msg := range gossiper.msgChnPriv{
			if strings.Compare(msg.Text,"") == 0 || strings.Compare(msg.Dest,"") == 0{ // in case someone does not include Dest or Test
				return
			}
			if strings.Compare(msg.Origin,"") == 0{
				msg.Origin = gossiper.origin
			}
			if strings.Compare(msg.Dest, gossiper.origin) == 0{
				fmt.Print("PRIVATE: ",msg.Origin,":",msg.HopLimit,":",msg.Text,"\n")
				gossiper.mupriv.Lock()
				gossiper.privateList[time.Now()] = msg
				gossiper.mupriv.Unlock()
			}else{
				msg.HopLimit--
				if msg.HopLimit == 0 {
					return
				}else{
					go gossiper.sendPrivateMessages(msg)
				}
			}
		}
	}()
}

func(gossiper *Gossiper)sendPrivateMessages(msg messaging.PrivateMessage) {
	sendTo := gossiper.Routerr.SearchRoutingTable(msg.Dest)
	if strings.Compare(sendTo,"") == 0{
		return
	}
	messageToSend := messaging.GossipPacket{nil,nil,&msg}
	gossiper.sendMessages(messageToSend, sendTo)
}

func (gossiper *Gossiper)announceMyself(){
	gossiper.msgID++
	rumor := messaging.RumorMessage{gossiper.origin, gossiper.msgID, "",gossiper.ip,gossiper.port}
	gossiper.storeVectorClocks(messaging.GossipPacket{&rumor,nil,nil})
	gossiper.sendRumors(rumor, gossiper.setpeers)
	for {
		ticker := time.NewTicker(60*time.Second)
		<-ticker.C
		gossiper.msgID++
		rumor := messaging.RumorMessage{gossiper.origin, gossiper.msgID, "",gossiper.ip,gossiper.port}
		gossiper.storeVectorClocks(messaging.GossipPacket{&rumor,nil,nil})
		// maybe send to only a random peer
		gossiper.sendRumors(rumor, gossiper.setpeers)

	}
}

func(gossiper *Gossiper)createStatusPacket() []messaging.PeerStatus{
	peerStatus := []messaging.PeerStatus{}
	var temp messaging.PeerStatus
	mapping := make(map[string]messaging.PeerStatus)
	for k, _ := range gossiper.vectorClock{
		origin :=strings.Split(k,",")[0]
		_, ok := mapping[origin]
		if !ok {
			temp = messaging.PeerStatus{origin,gossiper.lastRecordToReturn(origin)+1}
			mapping[origin] = temp
		}
	}
	for _, temp := range mapping{
		peerStatus = append(peerStatus, temp)
	}
	return peerStatus
}

func(gossiper *Gossiper)flipcoin(msg messaging.GossipPacket, relayPeer string){
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	num := r.Int31n(10)%2
	if num == 0 {
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		var temp int32
		temp = int32(len(gossiper.setpeers))
		num := r.Int31n(10)%temp
		randomPeer := gossiper.setpeers[num]
		if strings.Compare(randomPeer, relayPeer) == 0{
			if len(gossiper.setpeers) == 1{
				return
			}else{
				for{
					r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
					num = r.Int31n(10)%temp
					randomPeer = gossiper.setpeers[num]
					if strings.Compare(randomPeer, relayPeer) != 0{
							break
					}
				}
			}
		}
		fmt.Println("FLIPPED COIN sending rumor to",randomPeer)
		gossiper.forwardMessage(msg, relayPeer, randomPeer)
	}else{
		return
	}
}

func (gossiper *Gossiper)AntiEntropy(){
		for{
			ticker := time.NewTicker(1*time.Second)
			<-ticker.C
			if len(gossiper.vectorClock) >0 {
				gossiper.sendAntiEntropyMessage()
			}
	   }
}

func (gossiper *Gossiper)sendAntiEntropyMessage(){
	if len(gossiper.setpeers) > 0{
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		var temp int32
		temp = int32(len(gossiper.setpeers))
		num := r.Int31n(10)%temp
		randomPeer := gossiper.setpeers[num]
		toSend := messaging.GossipPacket{nil,&messaging.StatusPacket{gossiper.createStatusPacket()}, nil}
		go gossiper.sendMessages(toSend, randomPeer)
	}
	return
}

func (gossiper *Gossiper)lastRecordToReturn(key string) uint32{
	max := -1
	gossiper.mu.RLock()
	for k, _ := range gossiper.vectorClock{
		if strings.Compare(strings.Split(k,",")[0],strings.Split(key, ",")[0]) == 0{
			strtemp := strings.Split(k,",")[1]
			temp, _ := strconv.Atoi(strtemp)
			if temp > max{
				max = temp
			}
		}
	}
	gossiper.mu.RUnlock()
	return uint32(max)
}

func (gossiper *Gossiper)compareVectorClocks(vc messaging.MsgReceiver) bool{
	flag := true
	once := true
	peersExistent := make(map[string]bool)
	for index := range vc.Msg.Status.Want {
		str := vc.Msg.Status.Want[index].Identifier + "," + fmt.Sprint(vc.Msg.Status.Want[index].NextID)
		peersExistent[vc.Msg.Status.Want[index].Identifier] = true
		if gossiper.ifMessageExists(vc.Msg.Status.Want[index].Identifier, vc.Msg.Status.Want[index].NextID){
			flag = false
			temp, _ := gossiper.accessVectorClock(str)//gossiper.vectorClock[str]
			newPacket := messaging.GossipPacket{&messaging.RumorMessage{vc.Msg.Status.Want[index].Identifier,vc.Msg.Status.Want[index].NextID,temp, gossiper.ip, gossiper.port}, nil, nil}
			gossiper.sendMessages(newPacket, vc.UpdAddr)
		}else{
			if once{
				str1 := vc.Msg.Status.Want[index].Identifier+ ","+fmt.Sprint(vc.Msg.Status.Want[index].NextID-1)
				_, ok := gossiper.accessVectorClock(str1)//gossiper.vectorClock[str1]
				if !ok {
					once = false
					flag = false
					newPacket := messaging.GossipPacket{nil, &messaging.StatusPacket{gossiper.createStatusPacket()}, nil}
					gossiper.sendMessages(newPacket, vc.UpdAddr)
				}
			}
		}
	}
	for index := range vc.Msg.Status.Want {
		str := vc.Msg.Status.Want[index].Identifier + "," + fmt.Sprint(vc.Msg.Status.Want[index].NextID+1)
		if gossiper.ifMessageExists(vc.Msg.Status.Want[index].Identifier, vc.Msg.Status.Want[index].NextID+1){
			flag = false
			temp, _ := gossiper.accessVectorClock(str)//gossiper.vectorClock[str]
			newPacket := messaging.GossipPacket{&messaging.RumorMessage{vc.Msg.Status.Want[index].Identifier,vc.Msg.Status.Want[index].NextID,temp, gossiper.ip,gossiper.port}, nil, nil}
			gossiper.sendMessages(newPacket, vc.UpdAddr)
		}
	}
	gossiper.mu.RLock()
	for key, value := range gossiper.vectorClock{
		orig := strings.Split(key,",")[0]
		_, ok := peersExistent[orig]
		if !ok{
			flag = false
			temp,_ := strconv.Atoi(strings.Split(key,",")[1])
			newPacket := messaging.GossipPacket{&messaging.RumorMessage{orig,uint32(temp),value, gossiper.ip, gossiper.port}, nil, nil}
			gossiper.sendMessages(newPacket, vc.UpdAddr)
		}
	}
	gossiper.mu.RUnlock()
	if flag{
		fmt.Println("IN SYNC WITH",vc.UpdAddr)//, vc.Msg.Status)
	}
	return flag
	/*if flag{
		fmt.Println("IN SYNC WITH",vc.UpdAddr, vc.Msg.Status)
		go gossiper.flipcoin(vc.Msg, vc.UpdAddr)
		return
	}*/
}

func (gossiper Gossiper)storeVectorClocks(vc messaging.GossipPacket){
		str := vc.Rumor.Origin +","+ fmt.Sprint(vc.Rumor.ID)
		gossiper.mu.Lock()
		gossiper.vectorClock[str] = vc.Rumor.Text
		gossiper.timeofMessage[str] = time.Now()
		gossiper.mu.Unlock()
}

func (gossiper *Gossiper)appendPeersList(peerToAdd string){
	ok := false
	for _, value := range gossiper.setpeers {
		if strings.Compare(value, peerToAdd) == 0 {
			ok = true
			break
		}
	}
	if !ok {
		gossiper.setpeers = append(gossiper.setpeers, peerToAdd)
	}
}

func (g *Gossiper)sendMessages(msg messaging.GossipPacket, peer string){
	var message *messaging.GossipPacket
	message = &msg
	packetBytes, err := protobuf.Encode(message)
	if err != nil{
		log.Fatal("Error5: ", err)
	}
	pdAddr, _ := net.ResolveUDPAddr("udp4", peer)
	g.conn.WriteToUDP(packetBytes, pdAddr)
}

func (gossiper *Gossiper)checkIfClient(msg messaging.GossipPacket) bool{
	if msg.Rumor != nil{
		if msg.Rumor.Origin == "" {
			return true
		}
	}
	return false
}

// return true for RumorMessage, else false for StatusPacket
func (gossiper *Gossiper)checkMessageType(msg messaging.GossipPacket) bool{
	if msg.Rumor != nil{
		return true
	}
	return false
}

func (gossiper * Gossiper)printKnownPeers(){
	i := 0
	for _,k := range gossiper.setpeers{
		i++
		fmt.Print(k)
		if i < len(gossiper.setpeers) {
			fmt.Print(",")
		}
	}
	fmt.Println()
}

func (gossiper *Gossiper)ifMessageExists(origin string, ID uint32) bool {
	str := origin+","+fmt.Sprint(ID)
	_, ok := gossiper.accessVectorClock(str)//gossiper.vectorClock[str]
	if !ok {
		return false
	}else{
		return true
	}
}

func (gossiper *Gossiper)accessVectorClock(key string) (string, bool){
	gossiper.mu.RLock()
	v, ok := gossiper.vectorClock[key]
	defer gossiper.mu.RUnlock()
	return v, ok
}

func (gossiper * Gossiper)printVectorClock(){
	i := 0
	fmt.Println(gossiper.origin,len(gossiper.vectorClock))
	for n,k := range gossiper.vectorClock{
		i++
		fmt.Println("Vector clock of",n)
		for _,pin := range k{
			fmt.Print(pin)
		}
	}
	fmt.Println()
}