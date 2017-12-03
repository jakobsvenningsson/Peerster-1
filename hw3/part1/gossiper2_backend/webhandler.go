package gossiper2_backend

import (
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"log"
	"fmt"
	"strings"
	"github.com/sagap/Peerster/hw3/part1/messaging"
)


type myHandler struct{
	node    *Gossiper
	//proposeC   <-chan(string)
}

type acceptIP struct{
	IP string
}

func (gossiper *Gossiper) nodeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":                 // add a new neighbor node
		var m acceptIP
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &m)
		gossiper.setpeers = append(gossiper.setpeers, m.IP)
		w.WriteHeader(http.StatusOK)
	case "GET":                      // return the list of neighbor nodes(peers)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		j, _ := json.Marshal(gossiper.setpeers)
		w.Write(j)
	}
}

func (gossiper *Gossiper) nodeIDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":                      // return the list of neighbor nodes(peers)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		j, _ := json.Marshal(gossiper.Routerr.PrintOriginIdentifiers())       // returns nodeIDs
		w.Write(j)
	}
}

func (gossiper *Gossiper) messageHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case "POST":
		var m messaging.GossipPacket
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &m)
		gossiper.messageFromGUI(m)
		w.WriteHeader(http.StatusOK)
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		toPrint := []string{}
		//gossiper.mu.RLock()
		// Vector Clocks are used well, but I did not put any logic on the way messages should be printed
		/*for k, v := range gossiper.vectorClock{
			if strings.Compare(v,"") != 0{
				toPrint = append(toPrint, gossiper.timeofMessage[k].String()+" : "+strings.Split(k,",")[0]+" -> "+v)
			}
		}*/
		for _, val := range gossiper.MessagesText(){
			toPrint = append(toPrint, val)
		}
		//gossiper.mu.RUnlock()
		// according to specs there's no need to keep PRIVATE messages stored
		for k, v := range gossiper.privateList{
			toPrint = append(toPrint,"PRIVATE: "+k.String()+" : from "+v.Origin+" -> "+v.Text)
			fmt.Println("EDWWWWWWWWWWWW",v.Text)
			delete(gossiper.privateList, k)
		}
		j, _ := json.Marshal(toPrint)
		w.Write(j)
	}
}

func (gossiper *Gossiper) changeIDHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	var m string
	b, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(b, &m)
	//gossiper.ChangeID(m)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	j, _ := json.Marshal(gossiper.origin)
	w.Write(j)
}

func (gossiper *Gossiper)ServeHttpKVAPI() {
	r := mux.NewRouter()
	r.HandleFunc("/message", gossiper.messageHandler).Methods("GET", "POST")
	r.HandleFunc("/node", gossiper.nodeHandler).Methods("GET", "POST")
	r.HandleFunc("/nodeId", gossiper.nodeIDHandler).Methods("GET")
	r.HandleFunc("/id", gossiper.changeIDHandler).Methods("POST")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static"))))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(gossiper.GuiPort),r))//gossiper.GuiPort), r))
}

func (gossiper *Gossiper)messageFromGUI(packet messaging.GossipPacket){
	if packet.Rumor != nil{
		gossiper.msgChn1 <- packet
	}else if packet.Private != nil{
		var newPrivateMessage messaging.PrivateMessage
		newPrivateMessage.HopLimit = 10
		newPrivateMessage.Origin = gossiper.origin
		newPrivateMessage.Text = packet.Private.Text
		newPrivateMessage.Dest = strings.TrimSpace(packet.Private.Dest)
		newPrivateMessage.ID = 0
		gossiper.msgChnPriv <- newPrivateMessage
	}
}