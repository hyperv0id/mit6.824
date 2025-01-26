package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

	ok := true
	for ok {
		pid := os.Getpid()
		args := TaskArgs{pid}
		reply := TaskReply{}
		ok = call("Coordinator.DispatchTask", &args, &reply)
		if !ok {
			log.Printf("call failed!\n")
		}
		log.Printf("%d got task %d type %v\n", pid, reply.TaskID, reply.TaskType)

		var err error
		switch reply.TaskType {
		case MapTask:
			{
				err = do_map(reply, mapf)
			}
		case ReduceTask:
			{
				err = do_reduce(reply, reducef)
			}
		case WaitTask:
			{
				time.Sleep(time.Millisecond * 500) // sleep for 500ms
				continue
			}
		case DoneTask:
			{
				os.Exit(0)
			}
		}
		done(reply.TaskID, reply.TaskType, err)
	}
}

// 任务完成，报告状态
func done(wid int, taskType TaskType, err error) {
	log.Printf("%d done type %d\n", wid, taskType)
	args := FinishArgs{wid, taskType, err}
	reply := TaskReply{}
	ok := call("Coordinator.FinishTask", &args, &reply)
	if !ok {
		log.Printf("%d call FinishTask failed!\n", wid)
	}
}

// 执行map逻辑，读文件并分块保存到 mr-out-{taskID}-{reduceID}.txt 中
func do_map(reply TaskReply, mapf func(string, string) []KeyValue) error {
	log.Printf("%d map %v\n", reply.TaskID, reply.InputFiles)
	intermediate := []KeyValue{}
	for _, fn := range reply.InputFiles {
		file, err := os.Open(fn)
		if err != nil {
			return fmt.Errorf("cannot open %v: %v", fn, err)
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			file.Close()
			return fmt.Errorf("cannot read %v: %v", fn, err)
		}
		file.Close()
		kva := mapf(fn, string(content))
		intermediate = append(intermediate, kva...)
	}

	// Create temp files
	dir, _ := os.Getwd()
	mapfiles := []*os.File{}
	for i := 0; i < reply.NumReduce; i++ {
		oname, err := os.CreateTemp(dir, "")
		if err != nil {
			return fmt.Errorf("cannot create temp file: %v", err)
		}
		mapfiles = append(mapfiles, oname)
	}

	// Write intermediate key-value pairs to temp files
	for _, kv := range intermediate {
		idx := ihash(kv.Key) % reply.NumReduce
		file := mapfiles[idx]
		enc := json.NewEncoder(file)
		if err := enc.Encode(&kv); err != nil {
			return fmt.Errorf("cannot encode key-value pair: %v", err)
		}
	}

	// Close all files before renaming
	for _, v := range mapfiles {
		v.Close()
	}

	// Rename temp files to final names
	for i, v := range mapfiles {
		if err := os.Rename(v.Name(), fmt.Sprintf("mr-%d-%d", reply.TaskID, i)); err != nil {
			log.Printf("cannot rename file: %v\n", err)
		}
	}

	return nil
}

// 执行reduce逻辑，
func do_reduce(reply TaskReply, reducef func(string, []string) string) error {
	log.Printf("%d reduce %v\n", reply.TaskID, reply.InputFiles)
	kva := []KeyValue{}
	for _, fn := range reply.InputFiles {
		file, err := os.Open(fn)
		if err != nil {
			log.Fatalf("cannot open %v", fn)
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva = append(kva, kv)
		}
		file.Close()
	}

	//shuffle
	sort.Sort(ByKey(kva))
	dir, _ := os.Getwd()
	oname, _ := os.CreateTemp(dir, "")
	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[j].Key == kva[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(oname, "%v %v\n", kva[i].Key, output)
		// fmt.Printf("%v %v\n", kva[i].Key, output)
		i = j
	}
	oname.Close()
	os.Rename(oname.Name(), fmt.Sprintf("mr-out-%d", reply.TaskID))
	return nil
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		log.Printf("reply.Y %v\n", reply.Y)
	} else {
		log.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	log.Println(err)
	return false
}
