package routing

import (
	"strings"
	"fmt"
	"github.com/sagap/Decentralized-Systems-Project-2/part2/routingtable"
	"github.com/sagap/Decentralized-Systems-Project-2/part2/messaging"
	"strconv"
	//"time"
)

type Router struct{
	routingtable 			routingtable.RoutingTable
	lastSequencePerOrigin 	map[string]uint32
}

func NewRouter(rumor <-chan messaging.RumorMessage) (*Router){
	rr :=  &Router{
		routingtable: 			*routingtable.NewRoutingTable(),
		lastSequencePerOrigin:  make(map[string]uint32),
	}
	go rr.waitForRumorMessage(rumor)
	return rr
}

func (router *Router) waitForRumorMessage(rumor <-chan messaging.RumorMessage){

	for data := range rumor{
		if router.checkSeqNumber(data.Origin, data.ID){
			router.routingtable.Add(data.Origin, data.LastIP.String()+":"+strconv.Itoa(*data.LastPort))
		}
		//ticker := time.NewTicker(time.Second)
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

func (router *Router)Print(){
	fmt.Println("WPA1:",router.routingtable,"\t","WPA2:",router.lastSequencePerOrigin,"\n")
}

func (router *Router)PrintOriginIdentifiers() []string{
	origins := make([]string, 0, len(router.lastSequencePerOrigin))
	for k := range router.lastSequencePerOrigin {
		origins = append(origins, k)
	}
	return origins

}