package routing

import (
	"strings"
	"github.com/sagap/Peerster/hw3/part1/routingtable"
	"github.com/sagap/Peerster/hw3/part1/messaging"
	"strconv"
	//"time"
	//"time"
)

type Router struct{
	routingtable 			routingtable.RoutingTable		// routing table
	lastSequencePerOrigin 	map[string]uint32               // maintains last sequence number of each nodeID
	direct		map[string]bool
}

type PrioritySequence struct {
	Message messaging.RumorMessage
	Direct  bool                        // if it is a direct route
}

func NewRouter(rumor <-chan PrioritySequence, wait int) (*Router){
	rr :=  &Router{
		routingtable: 			*routingtable.NewRoutingTable(),
		lastSequencePerOrigin:  make(map[string]uint32),
	}
	go rr.waitForRumorMessage(rumor, wait)
	return rr
}

func (router *Router) waitForRumorMessage(rumor <-chan PrioritySequence, rtimer int){
	for data := range rumor{
		if router.checkSeqNumber(data.Message.Origin, data.Message.ID){
			if *data.Message.LastPort !=0{
				router.routingtable.Add(data.Message.Origin, data.Message.LastIP.String()+":"+strconv.Itoa(*data.Message.LastPort))
			}
		}else if data.Direct && data.Message.ID == router.lastSequencePerOrigin[data.Message.Origin]{
			router.routingtable.Add(data.Message.Origin, data.Message.LastIP.String()+":"+strconv.Itoa(*data.Message.LastPort))
		}

		//ticker := time.NewTicker(time.Duration(rtimer)*time.Second)
		//<-ticker.C
	}
}

func (router *Router)SearchRoutingTable(origin string) string{
	return router.routingtable.Lookup(origin)
}

func (router *Router)checkSeqNumber(origin string, seq uint32) bool{
	val, ok := router.lastSequencePerOrigin[origin]
	if !ok{
		router.lastSequencePerOrigin[origin] = seq
		return true
	}else{
		if val < seq {
			router.lastSequencePerOrigin[origin] = seq
			return true
		}else{
			return false
		}
	}
}

func (router *Router)CheckIfRouteRumor(msg messaging.RumorMessage) bool{
		if strings.Compare(msg.Text,"") == 0 {
			return true
		}
	return false
}

func (router *Router)PrintOriginIdentifiers() []string{
	origins := make([]string, 0, len(router.lastSequencePerOrigin))
	for k := range router.lastSequencePerOrigin {
		origins = append(origins, k)
	}
	return origins

}
