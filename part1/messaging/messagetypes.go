package messaging

import (
	"fmt"
	"net"
)

// GossipPacket defines the packet that gets encoded or deserialized from the
// network.
type GossipPacket struct {
	Rumor   *RumorMessage
	Status  *StatusPacket
	Private *PrivateMessage
}

// RumorMessage denotes of an actual message originating from a given Peer in the network.
type RumorMessage struct {
	Origin   string
	ID       uint32
	Text     string
	LastIP   *net.IP
	LastPort *int
}

// StatusPacket is sent as a status of the current local state of messages seen
// so far. It can start a rumormongering process in the network.
type StatusPacket struct{
	Want []PeerStatus
}

type MsgReceiver struct{
	Msg 	GossipPacket
	UpdAddr string
}

// PrivateMessage is sent privately to one peer
type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Dest 		string
	HopLimit    uint32
}

//vector clock
type PeerStatus struct {
	Identifier string
	NextID uint32
}

func (g *GossipPacket)Print() {
	if g.Rumor != nil {
		fmt.Println(g.Rumor.Text, ",", g.Rumor.ID, ",",g.Rumor.Origin, ",",g.Rumor.LastIP, ",",*g.Rumor.LastPort)
	}else{
		for i := range g.Status.Want{
			fmt.Print(" origin ",g.Status.Want[i].Identifier," nextID ",g.Status.Want[i].NextID)
		}
		fmt.Println()
	}
}

func (pm *PrivateMessage)Print(){
	fmt.Println("Origin: ",pm.Origin,"Dest: ",pm.Dest,"HopLimit: ",pm.HopLimit,"ID: ",pm.ID,"Text: ",pm.Text)
}