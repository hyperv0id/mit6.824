package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"log"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (3A).
	return rf.currentTerm, atomic.LoadInt32(&rf.RoleState) == Leader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.killed() {
		reply.VoteGranted = false
		return
	}
	defer DPrintf("%d<Vote %d|%+v", rf.me, args.CandidateId, reply)
	// defer DPrintf("%d<Vote %d|%+v", rf.me, args.CandidateId, reply)
	if args.Term < rf.currentTerm {
		// TODO: 变成follower
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	if args.Term == rf.currentTerm {
		// 本任期已经投票给别人，不能再次投票
		if rf.voteFor != -1 && rf.voteFor != args.CandidateId {
			reply.Term = rf.currentTerm
			reply.VoteGranted = false
		}
	}
	if args.Term > rf.currentTerm {
		rf.becomeFollower()
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
		rf.voteFor = args.CandidateId
	} else {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
	}
	atomic.StoreInt32(&rf.shouldElect, 0)

}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	if server == rf.me {
		return false
	}
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer DPrintf("%d<AE %d -> %+v", rf.me, args.LeaderId, reply)
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}
	reply.Term = rf.currentTerm
	reply.Success = true
	rf.becomeFollower()
	rf.currentTerm = args.Term
	rf.voteFor = args.LeaderId
	atomic.StoreInt32(&rf.shouldElect, 0)

	// 1. 新term、自己是leader ==> follow
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) {
	if server == rf.me {
		return
	}
	DPrintf("%d>AE %d -> %+v", rf.me, server, args)
	rf.peers[server].Call("Raft.AppendEntries", args, reply)
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// 检查是否超时需要选举的代码
func (rf *Raft) ticker() {
	for !rf.killed() {
		// Your code here (3A)
		// Check if a leader election should be started.

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		time.Sleep(RandElectionTime())
		// 原子判断是否需要选举
		if atomic.LoadInt32(&rf.RoleState) != Leader && atomic.LoadInt32(&rf.shouldElect) == 1 {
			go rf.startElection()
		}
		atomic.StoreInt32(&rf.shouldElect, 1)
	}
}

// 开始选举
func (rf *Raft) startElection() {
	if rf.killed() {
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.currentTerm++
	rf.voteFor = rf.me
	rf.RoleState = Candidate
	// rf.voteCnt = 1
	voteCnt := 1
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.lastLogIndex(), // TODO
		LastLogTerm:  rf.lastLogTerm(),
	}
	DPrintf("%d StartElection %+v", rf.me, args)
	for i := range rf.peers {
		// 发送投票
		go func(pid int) {
			if pid == rf.me {
				return
			}
			var reply RequestVoteReply
			DPrintf("%d>Vote %d>%+v", rf.me, pid, args)
			ok := rf.sendRequestVote(pid, &args, &reply)
			DPrintf("%d>Vote %d<%+v\tvcnt=%d", rf.me, pid, reply, voteCnt)
			if !ok || !reply.VoteGranted || rf.killed() {
				return
			}
			voteCnt++
			if atomic.LoadInt32(&rf.RoleState) == Candidate && voteCnt > len(rf.peers)/2 {
				// 变成leader loop 仅一次
				go rf.becomeLeader()
			}
		}(i)
	}
}

// 领导者不断维持心跳的代码
func (rf *Raft) becomeLeader() {
	DPrintf("%d->Leader, term:%d", rf.me, rf.currentTerm)
	// rf.mu.Lock()
	rf.RoleState = Leader
	args := &AppendEntriesArgs{
		Term:     rf.currentTerm,
		LeaderId: rf.me,
	}
	// rf.mu.Unlock()
	for !rf.killed() && atomic.LoadInt32(&rf.RoleState) == Leader {
		// leader 负责周期性地广播发送AppendEntries rpc请求，candidate 也负责周期性地广播发送RequestVote rpc 请求
		for i := range rf.peers {
			go rf.sendAppendEntries(i, args, &AppendEntriesReply{})
		}
		time.Sleep(RandHeartBeatTime())
	}
}

// 变成Follower，需要相关变量归位
func (rf *Raft) becomeFollower() {
	rf.RoleState = Follower
	rf.voteFor = -1
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	log.SetFlags(0)
	DPrintf("Init: %d/%d", me, len(peers))
	rf := &Raft{
		mu:          sync.Mutex{},
		peers:       peers,
		persister:   persister,
		me:          me,
		dead:        0,
		logs:        make([]LogEntry, 0),
		nextIdx:     make([]int, 0),
		matchIdx:    make([]int, 0),
		commitIdx:   0,
		lastApplied: 0,
		currentTerm: 0,
		voteFor:     -1,
		RoleState:   Follower,
		shouldElect: 1, // 没有消息
	}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (3A, 3B, 3C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
