package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type KVServer struct {
	mu         sync.Mutex
	data       map[string]string
	replyCache map[int64]PutAppendReply
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// TODO: Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if value, ok := kv.data[args.Key]; ok {
		reply.Value = value
	} else {
		reply.Value = ""
	}
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// TODO: Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if cch, ok := kv.replyCache[args.ReqID]; ok {
		*reply = cch
	} else {
		key := args.Key
		kv.data[key] = args.Value
		kv.replyCache[args.ReqID] = *reply
		if args.ReqID > 0 {
			delete(kv.replyCache, args.ReqID-1)
		}
	}
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// TODO: Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if cachedReply, ok := kv.replyCache[args.ReqID]; ok {
		*reply = cachedReply
		return
	} else {
		key := args.Key
		reply.Value = kv.data[key]
		kv.data[key] += args.Value
		kv.replyCache[args.ReqID] = *reply
		if args.ReqID > 0 {
			delete(kv.replyCache, args.ReqID-1)
		}
	}

}

func StartKVServer() *KVServer {
	kv := &KVServer{
		mu:         sync.Mutex{},
		data:       make(map[string]string),
		replyCache: make(map[int64]PutAppendReply),
	}
	return kv
}
