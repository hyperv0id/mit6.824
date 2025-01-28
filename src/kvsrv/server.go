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
	mu sync.Mutex
	store map[string]string
	// Your definitions here.
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// TODO: Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	reply.Value = kv.store[args.Key]
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// TODO: Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store[args.Key] = args.Value
	reply.Value = kv.store[args.Key]
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// TODO: Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	reply.Value = kv.store[args.Key]
	kv.store[args.Key] += args.Value
}

func StartKVServer() *KVServer {
	kv := new(KVServer)
	kv.store = make(map[string]string)
	// TODO: You may need initialization code here.

	return kv
}
