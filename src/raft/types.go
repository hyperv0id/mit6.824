package raft

import (
	"sync"

	"6.5840/labrpc"
)

const (
	Follower int32 = iota
	Candidate
	Leader
)

type LogEntry struct {
	Term    int
	Command interface{}
}

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	logs        []LogEntry
	nextIdx     []int
	matchIdx    []int
	commitIdx   int
	lastApplied int
	currentTerm int
	voteFor     int
	// voteCnt         int
	RoleState       int32
	shouldElect int32 // 01代表ticker时间内有无消息
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
	// prevLogIndex int //index of log entry immediately preceding new ones
}
type AppendEntriesReply struct {
	Term    int
	Success bool // Append操作结果
}