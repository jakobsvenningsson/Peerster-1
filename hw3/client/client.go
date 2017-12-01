package main

import (
"flag"
"net"
"fmt"
"strconv"
"github.com/dedis/protobuf"
	"github.com/sagap/Peerster/hw3/messaging"
	"strings"
)
const localAddress string = "127.0.0.1"

func main(){
	uiPort := flag.Int("UIPort",10000,"user interface port(for clients)")
	dest := flag.String("Dest", "", "Destination")
	msg := flag.String("msg", "Hello", "Message to be sent")
	file := flag.String("file", "", "Share this file")
	request := flag.String ("request", "", "Hash Value of Metafile")
	flag.Parse()
	var message messaging.GossipPacket
	if strings.Compare(*file,"")!=0{
		datareq := &messaging.DataRequest{"",*dest,0,*file,[]byte(*request)}
		message = messaging.GossipPacket{nil,nil,nil,datareq,nil}
	}else if strings.Compare(*dest,"") != 0{
		privateMessage := &messaging.PrivateMessage{"",0,*msg,*dest,10}
		message = messaging.GossipPacket{nil,nil,privateMessage,nil,nil}
	}else{
		rumorMessage := &messaging.RumorMessage{"", 0,*msg, &net.IP{}, uiPort}
		message = messaging.GossipPacket{rumorMessage,nil,nil, nil,nil}
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

	//fmt.Println("Send: ", packetBytes,message.Private)
	conn.Write(packetBytes)
}