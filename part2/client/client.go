package main

import (
"flag"
"net"
"fmt"
"strconv"
"github.com/dedis/protobuf"
	"github.com/sagap/Peerster/part2/messaging"
	"strings"
)
const localAddress string = "127.0.0.1"

func main(){
	uiPort := flag.Int("UIPort",10000,"user interface port(for clients)")
	dest := flag.String("Dest", "", "Destination")
	msg := flag.String("msg", "Hello", "Message to be sent")
	flag.Parse()
	var message messaging.GossipPacket
	if strings.Compare(*dest,"") == 0{
		rumorMessage := &messaging.RumorMessage{"", 0,*msg, &net.IP{}, uiPort}
		message = messaging.GossipPacket{rumorMessage,nil,nil}
	}else{
		privateMessage := &messaging.PrivateMessage{"",0,*msg,*dest,10}
		message = messaging.GossipPacket{nil,nil,privateMessage}
	}
	toSend := localAddress+":"+strconv.Itoa(*uiPort)
	updAddr, err1 :=net.ResolveUDPAddr("udp",toSend)
	if err1 != nil{
		fmt.Println(err1)
	}
	conn, err := net.DialUDP("udp", nil, updAddr)
	if err != nil{
		fmt.Println(err)
	}
	defer conn.Close()
	packetBytes, err := protobuf.Encode(&message)
	//fmt.Println("Send: ", packetBytes,message.Rumor.Text)
	conn.Write(packetBytes)
}