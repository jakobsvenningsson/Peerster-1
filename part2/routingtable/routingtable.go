package routingtable

import (
	"sync"
	"fmt"
)

type RoutingTable struct{
	routingTable map[string]string
	mu    sync.RWMutex
}

func NewRoutingTable() *RoutingTable{
	routingTable := &RoutingTable{
		routingTable: make(map[string]string),
	}
	return routingTable
}

func (kv *RoutingTable) Contains(key string) (string, bool) {
	kv.mu.RLock()
	val, ok := kv.routingTable[key]
	kv.mu.RUnlock()
	return val, ok
}

func (kv *RoutingTable) Add(k, v string) {
	kv.mu.Lock()
	//val, _ := kv.routingTable[k]
	//if !ok {
	//fmt.Print("DSDV ",k,":",v,"\n")
	fmt.Printf("DSDV %s: %s\n", k, v)
	kv.routingTable[k] = v
	//}
	kv.mu.Unlock()
}

func (kv *RoutingTable)Lookup(key string) string{
	value, ok := kv.routingTable[key]
	if !ok{
		return ""
	}
	return value
}