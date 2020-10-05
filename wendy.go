package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
)

// Definition of message formats

// Generic Message (send by everyone).
type message struct {
	sender   int
	receiver int
	mtype    string
	time     int // Delivery time; this is actually part of the network
	content  string
}

// Transaction as send by traders (as content of a message)
type Transaction struct {
	Marketid int    `json:"MID"`
	Payload  string `json:"payl"`
}

type Block struct {
	Payload string `json:"payl"`
}

// Vote message
type Vote struct {
	SeqNumber    int
	Marketid     int    `json:"MID"`
	Payload      string `json:"payl"`
	ReceivedTime int
	Sender       int
}

// Validator State

// TransactionV as saved in the internal state of a validator
type TransactionV struct {
	Marketid     int    `json:"MID"`
	Payload      string `json:"payl"`
	ReceivedTime int
	SeqNumber    int
	blockers     []int  // Which transaction (by number) is blocking me
	Votes        []Vote // Who voted for this transaction
}

type validatorState struct {
	Id             int
	X_Coord        int // Coordinates to have a more realistic message delay estimate
	Y_Coord        int
	SequenceNumber int
	OtherSeqNos    [20]int // Last sequence number seen by others
	Transactions   []TransactionV
	LastDoneTX     int // Last Transaction auch that all earlier ones
	// have finished (needed to clear memory)
	IncomingQ [20][]message // Votes we can't process yet due to the wrong sequence number
	Br        [][]string    // Used to compute the blockings. For each transactions, store the blocking transactions
	Q         []string      // List of requests ready for the next block
	D         []string      // List of requests already dealt with
	U         []string      // List of all known requests not in Q or D

}

var worldTime int
var messageBuffer []message
var vd [20]validatorState // Max numvber of validators for now
// Debugging and controlling
var n int // The first validator is ID 1, not 0
var t int
var r int
var debugLeader int // The party we watch for debug messages
var blockchainDelay int
var blockchainRnd int
var msgDelay int
var msgRnd int
var delay int   // How many ticks to wait until a trader sends a new message
var runtime int // How long do we want the simulator to run ?
//Statistical Stuff
var totalSpread int
var totalVotes int
var maxSpread int
var totalTX int
var totalDelayedTX int
var delayed_insufficient_votesTX int

// Slice Tools

func RemoveIndexInt(s []int, index int) []int {
	return append(s[:index], s[index+1:]...)
}
func RemoveIndexStr(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}
func RemoveIndexMsg(s []message, index int) []message {
	if len(s) <= 1 {
		return nil
	}
	return append(s[:index], s[index+1:]...)
}
func RemoveIndexTX(s []TransactionV, index int) []TransactionV {
	if len(s) <= 1 {
		return nil
	}
	return append(s[:index], s[index+1:]...)
}
func Checkpoint(s string, code int) {
	// Debugging output
	if code == 7 {
		fmt.Println("Checkpoint ", s)
	}
}

func showVotes(id int) {
	lll := len(vd[id].Transactions) - 1
	for lll >= 0 {
		fmt.Println(vd[id].Transactions[lll].Votes)
		lll--
	}
}

func showTX(id int) {
	fmt.Println("Transaction memory of Validator ", id)
	lll := len(vd[id].Transactions) - 1
	for lll >= 0 {
		fmt.Println(vd[id].Transactions[lll])
		lll--
	}
}
func showALL(id int) {
	fmt.Println("Full memory of Validator ", id)
	fmt.Println(vd[id])
}

//*********************************************************************
//*
//* Network Simulation
//* Essentially, the network is an array of strings; tjhe index
//*  indicats at whot global time the string is arriving at its
//*  destination.
//* For now we're a bit wasteful, as we don't remove historic
//* entries
//* As for IDs, we reserve 1-100 for validators and 100+ for traders
//* ID 0 is used for a control message that simulates an internal
//* event (it can be reused for a trader though).
//*
//* Delivery times are set at (sort of) random when a message is send.
//*
//* The main loop of the code is the network, which ticks through
//* global time and activates the recipient of a message once
//* a message is delivered, as well as activating some traders to
//* send transactions.. That means that all validators are completely
//* reactive
//*********************************************************************

func sendMessage(payload string, mtype string, sender int, receiver int) {

	var distance int
	X_Coord := rand.Intn(100)
	Y_Coord := rand.Intn(100)
	time := worldTime + msgDelay // Deterministic version that assures no message
	// is out of order.
	if msgRnd > 0 {
		time += rand.Intn(msgRnd)
	}
	if sender < 20 {
		X_Coord = vd[sender].X_Coord
		Y_Coord = vd[sender].Y_Coord
	}

	xd := (X_Coord - vd[receiver].X_Coord)
	yd := (Y_Coord - vd[receiver].Y_Coord)
	distance = int(math.Sqrt((float64(xd*xd + yd*yd))))

	if distance == -1 {
		fmt.Println("OOps")
	}
	//time = worldTime+distance+msgRnd;
	time = worldTime + (distance*100)/msgDelay + msgRnd

	if sender == r {
		fmt.Println(r, " Sends ", payload, "to ", receiver, ". Will arrive at ", time)
	}
	if sender == 999999 {
		fmt.Println("Sending Message: ", mtype, payload, time, "from", sender, " to ", receiver)
	}
	// Messages to self take no time
	if sender == receiver {
		time = worldTime + 1
	}

	//This is a testroutine to create an order that is bad for fairness
	//Requires n=4, t=1, msgrnd=0, msgdelay=100, delay 9 and deactivation
	//of the location, and blocktime = 1100 (else the problem goes
	// away before wendy needs to act on it)
	// Turn on by settint the type back to "TX"
	//It's hard to be a nasty adversary here!
	if mtype == "TX_" {
		tmp := 0
		t2 := Transaction{}
		json.Unmarshal([]byte(payload), &t2)
		tmp_payload := t2.Payload
		json.Unmarshal([]byte(tmp_payload), &tmp)
		if tmp > 3700 { // For now, we only care about 4 transactions
			time += 20000
		} else {
			if receiver == 1 && tmp == 901 {
				time = 1100
			}
			if receiver == 1 && tmp == 1801 {
				time = 1200
			}
			if receiver == 1 && tmp == 2701 {
				time = 1300
			}
			if receiver == 1 && tmp == 3601 {
				time = 1400
			}

			if receiver == 2 && tmp == 901 {
				time = 1200
			}
			if receiver == 2 && tmp == 1801 {
				time = 1300
			}
			if receiver == 2 && tmp == 2701 {
				time = 1400
			}
			if receiver == 2 && tmp == 3601 {
				time = 1100
			}

			if receiver == 3 && tmp == 901 {
				time = 1300
			}
			if receiver == 3 && tmp == 1801 {
				time = 1400
			}
			if receiver == 3 && tmp == 2701 {
				time = 1100
			}
			if receiver == 3 && tmp == 3601 {
				time = 1200
			}

			if receiver == 4 && tmp == 901 {
				time = 1400
			}
			if receiver == 4 && tmp == 1801 {
				time = 1100
			}
			if receiver == 4 && tmp == 2701 {
				time = 1200
			}
			if receiver == 4 && tmp == 3601 {
				time = 1300
			}
		}

	}

	sendMessageWithTime(payload, mtype, sender, receiver, time)
}

func insertSorted(m message) {
	messageBuffer = append(messageBuffer, m)
	blen := len(messageBuffer) - 1
	if blen >= 0 {
		i := 0
		for m.time > messageBuffer[i].time && i <= blen+1 {
			i++
		} // for
		copy(messageBuffer[i+1:], messageBuffer[i:])
		messageBuffer[i] = m
	}
}

func sendMessageWithTime(payload string, mtype string, sender int, receiver int, time int) {
	m1 := message{}
	m1.sender = sender
	m1.time = time
	m1.receiver = receiver
	m1.content = payload
	m1.mtype = mtype
	//messagebuffer=append(messagebuffer,m1);
	insertSorted(m1)
}

func multicastMessageWithTime(payload string, mtype string, sender int, time int) {
	// Note: It is important to send a message to self as well for the
	// counting arguments to work!
	i := 0
	for i < n {
		sendMessageWithTime(payload, mtype, sender, i+1, time)
		i = i + 1
	}
}

func multicastMessage(payload string, mtype string, sender int) {
	// Note: It is important to send a message to self as well for the
	// counting arguments to work!
	i := 0

	for i < n {
		sendMessage(payload, mtype, sender, i+1)
		i = i + 1
	}
}

func generateTraderRequests() {
	// This function is called any tick.
	// We can send a trade per tick, or more or less
	t1 := &Transaction{Marketid: 5, Payload: "Test"}
	sender := rand.Intn(1000) + 100 // IDs 1-100 are reserved for valifdators
	//t1.Marketid = rand.Intn(5);
	t1.Marketid = 1

	temp, _ := json.Marshal(t1.Marketid + 100*worldTime)
	t1.Payload = string(temp)
	tenc, _ := json.Marshal(t1)
	payload := string(tenc)
	if (worldTime/delay)*delay == worldTime { // Slow things down a little for testing
		//fmt.Println("Sending TX",payload);
		multicastMessage(payload, "TX", sender)
	}
}

func processBlock(b string) { // Simulate the underlying blockchain, i.e.,
	// Once in a while pick up a block
	// We do this using the message interface

	Checkpoint("A block has been finished", 2)
	var leader = 1
	if n > 1 {
		leader = rand.Intn(n-1) + 1
	}
	leader = 1 // For testing purposes, we fix the leader.
	// TODO: This line should be deleted once we
	// are reasonably confident that there's no bugs left;
	// for now, it makes looking at the internal state easier
	// if we only need to watch one leader.

	// We simulate the blockchain via the network: essentially,
	// the leader sends a message with the block content to
	// all parties, and this scheduled the block (this message
	// is on it's way longer than normal ones as it simulates
	// the whole back and forth of the blockchain
	blockDelay := blockchainDelay
	if blockchainRnd > 0 {
		blockDelay += rand.Intn(blockchainRnd)
	}
	var q2, _ = json.Marshal(vd[leader].Q)
	fmt.Println("Proposing new block: ", len(vd[leader].Q), string(q2))
	totalTX += len(vd[leader].Q)
	fmt.Println("Number of TXs in block: ", len(vd[leader].Q), "(Total TX: ", totalTX, ")")

	totalDelayedTX += len(vd[leader].U)
	fmt.Println("Number of TXs delayed by Wendy: ", len(vd[leader].U), "(Total: ", totalDelayedTX, ")")
	ii := len(vd[leader].U) - 1
	count := 0
	for ii >= 0 {
		if isBlockedT(vd[leader].U[ii], leader) {
			count++
		}
		ii--
	}
	delayed_insufficient_votesTX += count
	fmt.Println("Of which ", count, "are blocked due to insufficient votes (Total:", delayed_insufficient_votesTX, ")")
	// We output the buffer only for one party, this is enough
	fmt.Println("Out of order votes in leader buffer: ", len(vd[leader].IncomingQ[2]))
	fmt.Println("Worldtime is:", worldTime)
	tmp := 1
	for tmp <= n {
		fmt.Println("Seq number of ", tmp, "is ", vd[leader].OtherSeqNos[tmp])
		tmp = tmp + 1
	}

	multicastMessageWithTime(string(q2), "Block", leader, worldTime+blockDelay)

	// This is a message we just send to the network so that
	// this function gets triggered again when the next block
	// can be processed. For Tendermint, this happens right after
	// the current block has arived. For concurrent blockchains
	// this could be faster than the multicast.
	sendMessageWithTime(string(q2), "BlockTrigger", 0, 0, worldTime+blockDelay+1)
}

//*************************************************************************************
//**
//** Actual Wendy Implementation
//**
//*************************************************************************************\

func isBlocked(m message, id int) bool {
	// If blocked means that it is possible that a transaction we haven't
	// even seen yet might require preference over the transaction in m.
	// This means we cannot process that request yet
	// So far we only implement the first scheme, where a message is blocked
	// if it has received less than t+1 votes.
	currentIndex := idByPayload(m.content, id)
	if len(vd[id].Transactions[currentIndex].Votes) > t {
		fmt.Println("Message ", m.content, " is unblocked with ", len(vd[id].Transactions[currentIndex].Votes), "votes.")
		return (false)
	}
	return true

}
func isBlockedT(s string, id int) bool {
	// If blocked means that it is possible that a transaction we haven't
	// even seen yet might require preference over the transaction in m.
	// This means we cannot process that request yet, or all that need to be
	// in the same block as it.
	var temp bool
	temp = true
	currentIndex := idByPayload(s, id)

	//fmt.Println("Just Checking",len(vd[id].Transactions[currentIndex].Votes), currentIndex )
	//fmt.Println(vd[id].Transactions[currentIndex].Votes,s,id)

	//fmt.Println(len(vd[id].Transactions[currentIndex].Votes),s,id)
	if len(vd[id].Transactions[currentIndex].Votes) > t {
		//return (false);
		temp = false
	}
	if id == debugLeader && temp {
		fmt.Println(s, " is blocked")
	}
	return (temp)
}

func seenEarlier(s1 string, s2 string, id1 int, id int) bool {
	// From the point of view of id, has id1 voted in a way that it has seen s1 before s2?
	// If they haven't been seen, we force in a false; this is because a 'true' value
	//  of seenEarleir is used to unblock transactions; if no information on the order
	//  is available, we can't do that.
	// If s1 has been seen and s2 has not, we need to return true
	index1 := idByPayload(s1, id)
	index2 := idByPayload(s2, id) //TODO: Can that be -1 ?
	seq1 := -1
	seq2 := -1

	// We now need all the votes id1 send to id, and then see which if s1 or s2 have
	// the vote with the lower sequence number.
	// So first, we take s1 (which is TX[index1]), and search for the vote id1 send
	// for it. seq 1 is then the seqeunce number id1 attached to that vote.
	i := len(vd[id].Transactions[index1].Votes) - 1
	for i >= 0 {
		if vd[id].Transactions[index1].Votes[i].Sender == id1 {
			seq1 = vd[id].Transactions[index1].Votes[i].SeqNumber
		}
		i--
	}

	i = len(vd[id].Transactions[index2].Votes) - 1
	for i >= 0 {
		if vd[id].Transactions[index2].Votes[i].Sender == id1 {
			seq2 = vd[id].Transactions[index2].Votes[i].SeqNumber
		}
		i--
	}

	//if (id==debugLeader) {
	//if(worldTime > 100) {
	//	fmt.Println("Sequence Numbers for sender ",id1," and votes ",s1,s2,": ",seq1,seq2);
	//}

	if seq2 == -1 && seq1 >= 0 {
		return (true)
	} // If s2 hasn't been seen yet, s1 had been seen earlier
	if seq1 == -1 && seq2 == -1 {
		return false
	}
	if seq1 == -1 {
		return (false)
	} // If s2 hasn't been seen either, it hasn't
	// This means that if neither has been seem, we return false;
	// If that is the case, how did we get here?
	if seq1 < seq2 {
		return (true)
	} // If both have been seen, their sequence number decides
	return (false)
}

func isBlocking(s1 string, s2 string, id int) bool {
	// If s1 blocking s2 (i.e., it needs to be in the same or an earlier block?
	//	s1 is blocking s2 if it is possible that all honest parties saw s1
	//	before s2. Thus, if >=t+1 parties voted s2 before s1, s1 is not blocking s2

	// We go through all ids and count seen_earlier(s2,s1...). If that id
	// hasn't seen s1 at all, it'll report false
	// If the two transactions have a different marketid,
	// then they're not blocking each other.
	//if vd[id].Transactions[idByPayload(s1, id)].Marketid != vd[id].Transactions[idByPayload(s2, id)].Marketid {
	//	return false
	//}
	//fmt.Println("Comparing ",s1,s2)
	i := n
	counter := 0
	//fmt.Println("SeenVotes: ",vd[id].Transactions[idByPayload(s1,id)].Votes," and ", vd[id].Transactions[idByPayload(s2,id)].Votes);
	for i >= 1 {
		if seenEarlier(s2, s1, i, id) {
			//fmt.Println(i," has seen ",s2," before ",s1);
			counter++
		}
		i--
	}
	//fmt.Println("Counter is",counter);
	// Counter now says how oftern s2 has been seen before s1.
	// If that's t+1 or more, we return false
	if id == debugLeader && counter < t+1 {
		fmt.Println(s1, " is blocking ", s2, " because ", counter)
	}
	return (counter < t+1)
}

func isDelivered(payload string, id int) bool {
	// Returns 'true' if payload has already been processed (is in D)
	i := len(vd[id].D) - 1
	for i > 0 {
		m := vd[id].D[i]
		if m == payload {
			return true
		}
		i = i - 1
	}
	return false
}

func seen(payload string, id int) bool {
	// Returns 'true' if Validator ID has already seen the transaction with payload 'Payload'
	i := len(vd[id].Transactions) - 1
	for i >= 0 {
		m := vd[id].Transactions[i].Payload
		if m == payload {
			return true
		}
		i = i - 1
	}
	return false
}

func indexByString(array []string, query string) int {
	i := len(array) - 1
	for i >= 0 {
		if array[i] == query {
			return i
		}
		i--
	}
	return -1
}

func idByPayload(payload string, id int) int {
	// Given the payload of a transaction, return it's position
	// in our internal datastructure so we can operate on it easier.
	i := len(vd[id].Transactions) - 1
	for i >= 0 {
		m := vd[id].Transactions[i].Payload
		if m == payload {
			return i
		}
		i = i - 1
	}
	return -1
}

func recompute(id int) {
	// This is a major part of Wendy. Each Validarot keeps a set of
	// transactions it cannot pass on to the blockchain yet because
	// of a fairness condition (U). What is in this set needs to be
	// reevaulated under two conditions
	//
	//  (1) a transaction in U has been delivered by the underlying blockchain
	//  (that would be because of another validator). This transaction now
	//  has been removed from the set of active transactions, and thus cannot
	//  block anyone else anymore. Thus, we need to check if any other transaction
	//  is now unblocked. Rather than messing around, we just recompute the blocking
	//  properties of all unfinished transactions
	//
	//  (2) a vote for a transaction came in that might unblock it. Same effect.
	//
	//   As we can unblock transactions here, there can be a cascade (this should be
	//   rare, but we don't want to exclude it) TODO: Verify if it's possible at all.
	//   Thus, if the recomputation moved a transaction from the active set into the
	//   delivery queue, we do the recomputation all over again. Thus will terminate
	//   as the active queue will eventually be empty (or nothig will get moved before
	//   that happens.
	//
	//    The main data structure is Br[][]. This is essentially a list that stores
	//    for each transaction what other transactions need to be in the same block
	//    to satisfy fairness. Every transaction is in it's own Br[][], and every
	//    transactin that blocks it is added.
	//    We then check for every transaction in Br[i][] if it is ready to be
	//    delivered (i.e., it cannot be blocked by a transaction we haven't seen yet)
	//    and then deliver the whole block (i.e., move all txs in there from U to Q.
	//    Afterwards, we recompute the Br[][] again, as some txs have disappeared now and
	//    might clear others.
	//
	//    Issues:
	//    Not the most efficient; we essentially recompute all Br[][] all the time.
	var finished bool
	var index int
	var j int

	//fmt.Println("RECOMPUTING",id)
	//showVotes(id);
	//fmt.Println("U: ", len(vd[id].U))
	finished = false
	for !finished {
		finished = true
		// Whenever the set of votes or unprocessed messages changes,
		// recompute all blocking sets B_r for all known and
		// unprocessed r. This might changr U, and thus be done
		// repeatedly.
		// All unprocessed messages are in U.

		// Now there has to be a much better way to do this; I need Br and
		// to_be_moved to have as many entries as U
		// Can I do to_be_moved[len(vd[id].U)] ??
		vd[id].Br = [][]string{}
		var toBeMoved []bool
		toBeMoved = []bool{}

		ii := len(vd[id].U) - 1
		for ii >= 0 {
			vd[id].Br = append(vd[id].Br, []string{}) // Create an entry for every entry in U
			toBeMoved = append(toBeMoved, false)
			ii--
		}

		i := len(vd[id].U) - 1
		for i >= 0 {
			vd[id].Br[i] = []string{vd[id].U[i]} // everyone is in their own blocking set
			j = len(vd[id].U) - 1
			for j >= 0 {
				if i != j && isBlocking(vd[id].U[j], vd[id].U[i], id) { // Don't add me to my blocking set twice
					vd[id].Br[i] = append(vd[id].Br[i], vd[id].U[j])
				}
				j--
			} // for j
			i--
		} // for i
		// Now, if a is blocked by b and b by c, a needs to be blocked by b too
		// So, we're not done yet

		if id == debugLeader {
			fmt.Println("Overview Br before filling in:")
			fmt.Println("------------------------------------")
			fmt.Println(vd[id].Br)
			fmt.Println("------------------------------------")
		}

		found_target := false
		var tmpstr string
		changed := true
		for changed {
			changed = false
			i = len(vd[id].Br) - 1
			for i >= 0 {
				j = len(vd[id].Br) - 1 // We loop through all B[i]s.
				for j >= 0 {           // and merge the B[j] of which B[i] contains B[j][0]
					k1 := len(vd[id].Br[i]) - 1 // So, first we loop through all elements
					// in  the currentB[i]
					for k1 >= 0 {
						if vd[id].Br[j][0] == vd[id].Br[i][k1] { // Our element in Bi matches the lead of Bj
							k2 := len(vd[id].Br[j]) - 1 // For all elements in Br[j]
							for k2 >= 0 {
								found_target = false
								k3 := len(vd[id].Br[i]) - 1
								tmpstr = vd[id].Br[j][k2]
								for k3 >= 0 { //... check if they're already in Br[i]
									if vd[id].Br[i][k3] == tmpstr {
										found_target = true
									}
									k3--
								}
								// If not, append to Br[i].
								if !found_target {
									vd[id].Br[i] = append(vd[id].Br[i], tmpstr)
									changed = true
								}
								k2--
							} //k2
						} // if
						k1--
					} //k1
					j--
				} //j
				i--
			} //i
		} // changed

		if id == debugLeader {
			fmt.Println("Overview Br at time", worldTime, ":")
			fmt.Println("------------------------------------")
			fmt.Println(vd[id].Br)
			fmt.Println("------------------------------------")
		}
		// Now, for every set Br, if any payload in Br is blocked, so is r
		// So we loop through all Brs, then through all payloads, and then
		// see if those are blocked.
		// If not, we move them from U to Q
		// We also need to move all entries in Br
		// fmt.Println("U: ", len(vd[id].U), "Q: ",len(vd[id].Q), " REC", len(vd[id].Br));
		// showVotes(id);

		var blocked bool
		i = len(vd[id].Br) - 1
		for i >= 0 {
			blocked = false
			j = len(vd[id].Br[i]) - 1
			for j >= 0 {
				if isBlockedT(vd[id].Br[i][j], id) {
					blocked = true
					if id == debugLeader {
						fmt.Println("Blockage for  ", i, " because ", j, vd[id].Br[i][j])
					}
				} //if
				j = j - 1
			} // for
			if !blocked {
				k := len(vd[id].Br[i]) - 1
				for k >= 0 {
					// Now I need to find the index back from the string
					index = indexByString(vd[id].U, vd[id].Br[i][k])
					if index > -1 {
						toBeMoved[index] = true
					}
					k = k - 1
				} //for
			} // if (!blocked)
			i = i - 1
		} //for
		i = len(vd[id].U) - 1
		for i >= 0 {
			if toBeMoved[i] {
				vd[id].Q = append(vd[id].Q, vd[id].Br[i][0]) // Br[i][0] = U[i] ?
				finished = false
				j = len(vd[id].U) - 1 // Now we need to look for the corresponding element
				// that we just put into Q in U to delete it.
				for j >= 0 {
					if vd[id].U[j] == vd[id].Br[i][0] {
						vd[id].U = RemoveIndexStr(vd[id].U, j) //TODO VERIFY

					} //if
					j = j - 1
				} //for
			} //if to me moved
			i--
		} //for i
	} // not finished
}

// One of Wendy's core functions: Get in a message and react on it.
// We also use this to monitor the blockchain
// What this function does is
//   * If we see a transaction first, store its data issue a vote
//   *  If the market_identifier of that tx is 0, it is pushed right to the blockchain
//      and not touched anymore.
//   * If we see a vote, count it
//   * If we see a block, remove all realated messages from the corresponding Queues
//   * If a vote comes in, call recompute to rebuild the internal data structures
//     and decide if a tx is through.
//
//  returns false if the message vouldn't be processed yet (i.e., it is a vote
//  with a too high sequence number/.

func processMessage(m message, id int) bool {
	// Function for Validator id to evaluate the incoming message m
	// This is the core of Wendy

	if id == debugLeader {
		fmt.Println(m)
	}
	if m.mtype == "Block" && id == 1 {
		//We use this for system output: A block has been delivered and we want to know
		//(this is sort of the App on validator 1 that we are watching).
		var q3 []string
		json.Unmarshal([]byte(m.content), &q3)
		fmt.Println("We've finished a block with ", len(q3), "entries")
		fmt.Println("-----------------------------------------------")
		i := len(q3) - 1
		totaltime := 0
		for i >= 0 {
			// Now would be a good time for some statistics
			value, _ := strconv.Atoi(q3[i])
			time := int(worldTime - (value / 100))
			fmt.Print("Msg ", i, ":", time, "     ")
			totaltime = totaltime + time
			totaltime = totaltime + 1
			i = i - 1
		}
		if len(q3) > 0 {
			fmt.Println("-------------------------------------")
			fmt.Println("Average time: ", totaltime/len(q3))

		}
		fmt.Println(q3)
	}

	if m.mtype == "Block" {
		// The underlying blockchain finished some
		// transactions. We have to delete them from
		// our active queues and put them into the
		// finished ones
		// i.e.,
		// Forall transactions T in BLOCK,
		//   FORALL Br, delete T from Br
		//   Delete T from Q
		//   Put T in D
		//   Then we need to go through processing
		//   just as if we got a vote.

		var q2 []string
		json.Unmarshal([]byte(m.content), &q2)
		i := len(q2) - 1
		for i >= 0 {
			// Append the transaction to our list of finished transactions
			vd[id].D = append(vd[id].D, q2[i])
			//Now we need to delete it from all our buffers
			// For now, we only delete it from Q and U
			// EXP: Also delete it from TX
			k := idByPayload(q2[i], id)
			if k > 0 {
				vd[id].Transactions = RemoveIndexTX(vd[id].Transactions, k)
			}
			j := len(vd[id].Q) - 1
			for j >= 0 {
				if vd[id].Q[j] == q2[i] {
					//fmt.Println("Scheduling Payload ",q2[i]);
					vd[id].Q = RemoveIndexStr(vd[id].Q, j)
				} //if
				j = j - 1
			} //for
			j = len(vd[id].U) - 1
			for j >= 0 {
				if vd[id].U[j] == q2[i] {
					//fmt.Println("Scheduling Payload ",q2[i]);
					vd[id].U = RemoveIndexStr(vd[id].U, j)
				} //if
				j = j - 1
			} //for
			i = i - 1
		}
		// After we removed all thransactions that block delivered
		// from the active set, we need to recompute all the blocks
		// and potentially put stuff from the active set to the queue
		// if one of the transactions we just removed was the one that
		// was blocking
		recompute(id)
	}

	if m.mtype == "TX" {
		// We got a new transaction
		// If the market identifier is 0, push
		// the transaction straight into Q. Or better,
		// maintain a second set for that, as we don't want
		// those transactions to cause a recompute
		// (Not that it matters here, but it matters for
		// the final computation. Actually, for the real thing,
		// we probaby want to redo the recompute thing
		// so that it only recomputes the affected market identifier.

		TX := TransactionV{}
		TX.ReceivedTime = worldTime
		TX.SeqNumber = vd[id].SequenceNumber
		t2 := Transaction{}
		json.Unmarshal([]byte(m.content), &t2)
		TX.Payload = t2.Payload
		//fmt.Println("Incoming Vote for validator",id, ": ", TX.Payload);
		TX.Marketid = t2.Marketid
		vote := Vote{}
		vote.ReceivedTime = TX.ReceivedTime
		vote.SeqNumber = TX.SeqNumber
		vote.Payload = TX.Payload
		vote.Marketid = TX.Marketid
		vote.Sender = id
		var m2, _ = json.Marshal(vote)

		//vd[id] is the state of the current validator
		if !seen(TX.Payload, id) {
			//fmt.Println("New Message");
			vd[id].SequenceNumber++
			vd[id].Transactions = append(vd[id].Transactions, TX)

			// If the MarketID is 0, that means the message needs no fairness
			// Thus, on seeing the transaction, we put it right into our output
			// Queue. For this simulation, we only do this when we receive the
			// Transaction, not on a vote (i.e., if a trader sends a message to only
			// a few traders, or has a slow connection to those traders, suchs to
			// be them.
			// We also don't vote on this message anymore, or do any checks for
			// dublicates here.
			if vote.Marketid == 0 {
				//fmt.Println("Appending ",TX.Payload," to leader ", id);
				vd[id].Q = append(vd[id].Q, TX.Payload)
			}
			if vote.Marketid != 0 {
				//fmt.Println("Appending ",TX.Payload," to unprocessed queue ", id);
				vd[id].U = append(vd[id].U, TX.Payload)
				if id == r {
					fmt.Println("# Saw TX, votes ", vote.SeqNumber)
				}
				multicastMessage(string(m2), "VOTE", id)
			}
			//fmt.Println("Queue is now ",len(vd[id].Q));

		} // seen
	} // TX

	if m.mtype == "VOTE" {
		// If the seqnos of the vote don't fit, we need to hold it back.
		vote := Vote{}
		TX := TransactionV{}
		json.Unmarshal([]byte(m.content), &vote)
		if 10 == debugLeader { //TODO: UGLY
			totalSpread = totalSpread + vote.SeqNumber - vd[id].OtherSeqNos[m.sender]
			totalVotes++
			if vote.SeqNumber-vd[id].OtherSeqNos[m.sender] > maxSpread {
				maxSpread = vote.SeqNumber - vd[id].OtherSeqNos[m.sender]
			}

			fmt.Println("Receive Vote seq.", vote.SeqNumber, " from P", m.sender, ". Was expecting ", vd[id].OtherSeqNos[m.sender]+1)
			fmt.Println("Avg Spread: ", totalSpread/totalVotes, "(", maxSpread, ")")
		}

		if vote.SeqNumber != 0 && vote.SeqNumber != vd[id].OtherSeqNos[m.sender]+1 {
			//fmt.Println("Message out of order from",m.sender,". Expecting ",vd[id].Other_Seq_Nos[m.sender]+1," got ",vote.Seq_Number);
			// If the message is out of order, we just return false; the caller
			// Todo: If the vote contains a new transaction, I can send that already
			return false
		} else {

			TX.Payload = vote.Payload
			vd[id].OtherSeqNos[m.sender] = vote.SeqNumber
			if !seen(vote.Payload, id) {
				myVote := Vote{0, vote.Marketid, vote.Payload, worldTime, id}
				myVote.SeqNumber = vd[id].SequenceNumber
				//vote.Sender = m.sender
				vd[id].SequenceNumber++
				TX.ReceivedTime = worldTime
				TX.SeqNumber = vote.SeqNumber //TODO ???
				//TX.voters=append(TX.voters,m.sender);
				vd[id].Transactions = append(vd[id].Transactions, TX)
				//vote.ReceivedTime = worldTime
				var m2, _ = json.Marshal(myVote)
				if vote.Marketid == 0 {
					//fmt.Println("Appending ",TX.Payload," to leader ", id);
					vd[id].Q = append(vd[id].Q, TX.Payload)
				}
				if vote.Marketid != 0 {
					//fmt.Println("Appending ",TX.Payload," to unprocessed queue ", id);
					vd[id].U = append(vd[id].U, TX.Payload)
					if id == r {
						fmt.Println("# Saw TX, votes ", vote.SeqNumber)
					}
					multicastMessage(string(m2), "VOTE", id)
				}
			} //seen
			// Manage votes
			// TODO: For production code, need to check for double votes
			// Also, maybe better to add a check if the vote was there already before
			// evaluating

			current_index := idByPayload(TX.Payload, id)
			if current_index > -1 {
				//vd[id].Transactions[current_index].voters = append(vd[id].Transactions[current_index].voters,m.sender);
				vd[id].Transactions[current_index].Votes = append(vd[id].Transactions[current_index].Votes, vote)
				if id == debugLeader {
					fmt.Println(id, " saw ", len(vd[id].Transactions[current_index].Votes), "votes for ", TX.Payload)
					fmt.Println("Sequence numbers are:")
					for iii := 0; iii < len(vd[id].Transactions[current_index].Votes); iii++ {
						fmt.Println(vd[id].Transactions[current_index].Votes[iii].Sender, vd[id].Transactions[current_index].Votes[iii].SeqNumber)
					} //for
					fmt.Println("U: ", len(vd[id].U), "Q: ", len(vd[id].Q), " REC", len(vd[id].Br))
				} //if
			}

			recompute(id)
		} // else (Vote has been processed)
	} //VOTE
	return true
}

func processIncomingQ(sender int, id int) {
	// Old function. Works, but was too slow.
	// Didn't want to delete it yet just in case.
	// The i queue contains all messages id received from sender that could not
	// be processed for now because they are voting messages with a future sequence number.
	// The last message here is that last added; if this one is out of order, all others
	// in the queue stay so. Otherwise, we replay the entire queue until it didn't
	// decrease in size anymore.
	// TODO: While we can get away with this due to the relatively small size of the queue,
	//       this starts getting horribly inefficient if the network hs a lot of randomness
	//       and we send a lot of transactions. A better way would be to use some sorted
	//       datastucture, so we can just resend messages from the front until the first one
	//       is out of sequence.
	//	As go doesn't seem to directly support sorting slices of structs according to one
	//	element in the struct, the easiest way would be to have the sequence number be
	//	the first element in the struct, convert it to a string, store the strings in
	//	Incoming_Q[][], use go sort, and convert it back. Which is ugly. Look for a
	//	better solution before doing that :)
	i := len(vd[id].IncomingQ[sender]) - 1
	qlen := i
	if processMessage(vd[id].IncomingQ[sender][i], id) {
		vd[id].IncomingQ[sender] = RemoveIndexMsg(vd[id].IncomingQ[sender], i)
		j := i - 1
		for j < qlen {
			qlen = len(vd[id].IncomingQ[sender]) - 1
			for j >= 0 {
				if processMessage(vd[id].IncomingQ[sender][j], id) {
					vd[id].IncomingQ[sender] = RemoveIndexMsg(vd[id].IncomingQ[sender], j)
				} // if
				j--
			} // for
			j = len(vd[id].IncomingQ[sender]) - 1
		}
	} // if
}

func seq(m message) int {
	var v Vote
	if m.mtype != "VOTE" {
		return -1
	}
	_ = json.Unmarshal([]byte(m.content), &v)
	return v.SeqNumber

}

func processIncomingQNew(m message) {
	// The i queue contains all messages id received from sender that could not
	// be processed for now because they are voting messages with a future sequence number.
	// The last message here is that last added; if this one is out of order, all others
	// in the queue stay so. Otherwise, we replay the entire queue until it didn't
	//iii:=0
	id := m.receiver
	qlen := len(vd[id].IncomingQ[m.sender]) - 1
	qlen2 := len(vd[id].IncomingQ[m.sender]) - 1
	// First, we process the new message..
	// if it goes through, we can eat up other messages in the queue
	// and don't need to sort.
	// Else, it is no part of the queue and we need to resort.
	if processMessage(m, m.receiver) {
		j := 0
		for j <= qlen2 {
			if processMessage(vd[id].IncomingQ[m.sender][j], id) {
				j++
				//fmt.Println("Processing Seq:",seq(vd[id].Incoming_Q[m.sender][j-1]));
			} else {
				qlen2 = -1
			}
			//fmt.Println("Before", len(vd[id].Incoming_Q[m.sender]));
			//fmt.Println("J was",j);
		} //for j<qlen2
		// Now we have processed a number of transactions, and we need to
		// delete them from the message list
		if j > 0 {
			vd[id].IncomingQ[m.sender] = append(vd[id].IncomingQ[m.sender][:0], vd[id].IncomingQ[m.sender][j:]...)
		}

		// If we didn't process the new element successfully, we need to insert
		// it at its right position in the incoming queue. No other element
		// in that queue needs processing, as none of them can have gotten
		// unblocked.
	} else {
		vd[id].IncomingQ[m.sender] = append(vd[id].IncomingQ[m.sender], m)
		if qlen >= 0 {
			i := 0
			for seq(m) > seq(vd[id].IncomingQ[m.sender][i]) && i <= qlen+1 {
				i++
			} // for
			copy(vd[id].IncomingQ[m.sender][i+1:], vd[id].IncomingQ[m.sender][i:])
			vd[id].IncomingQ[m.sender][i] = m
		}
	}
}

//********************************************************************************
//**
//** Wendy stops here.
//**
//********************************************************************************

// WIP: Clean the buffers of messages that have already been processed
// Incompatible with lastmsg
// Things that also could be cleaned: TX Buffer
func cleanMemory() {
	i := 0
	for i < len(messageBuffer)-1 && messageBuffer[i].time < worldTime {
		i++
	}
	if i > 0 {
		messageBuffer = append(messageBuffer[:0], messageBuffer[i:]...)
	}
}

func network() {
	// Network simulator. This is the old version that doesn't sort messages by time
	// Essentially, we just add one tick and see if there's an undelivered
	// for that time, and then process it. There is a special message type
	// BlockTrigger used to simulate the underlying blockchains (i.e., trigger
	// the processing of a new block).

	for worldTime < runtime || runtime == 0 {
		worldTime = worldTime + 1
		lastmsg := 0
		lastmsg2 := 0
		//fmt.Println("The time is :", worldTime)
		generateTraderRequests()
		i := len(messageBuffer) - 1 //TODO: Needs optimizing :)
		lastmsg = lastmsg2          //Since e're counting down, no need
		if lastmsg < 0 {
			lastmsg = 0
		} //to check smaller entries than the last
		//successfull one from last time.
		for i >= lastmsg {
			m := messageBuffer[i]
			if m.time > worldTime {
				lastmsg2 = i
			} // lastmsg2 now should be the smallest i with a message still to be delivered
			// so everything in the messagebuffer smaller than lastmsg2 can be ignored from now
			// on. For a more serious implementation, we should use some form of ringbuffer
			// here, but for our purposes this will do.

			if m.time == worldTime {
				//fmt.Println("The message is ",m.time,"  ",m.content);
				if m.mtype == "VOTE" {
					processIncomingQNew(m)
					//if (m.sender < 20) {
					//vd[m.receiver].Incoming_Q[m.sender] = append(vd[m.receiver].Incoming_Q[m.sender],m)
					//process_incoming_Q(m.sender,m.receiver);
				} else {
					_ = processMessage(m, m.receiver)
				}
				//processMessage(m,m.receiver);
				if m.mtype == "BlockTrigger" {
					processBlock(m.content)
				}
			}
			i = i - 1
		}
	}
}
func networkNew() {
	// Network simulator.
	// Essentially, we just add one tick and see if there's an undelivered
	// for that time, and then process it. There is a special message type
	// BlockTrigger used to simulate the underlying blockchains (i.e., trigger
	// the processing of a new block).
	// TODO: Just like the Incoming_Q, I could sort the messagebuffer.
	// Not sure yet if that's worth the effort though.
	lastmsg := 0
	for worldTime < runtime || runtime == 0 {
		if lastmsg > 1 {
			//fmt.Println("Cleaning buffer. Size was ",len(messageBuffer));
			messageBuffer = append(messageBuffer[:0], messageBuffer[lastmsg:]...)
			//fmt.Println("Cleaning buffer. Size is ",len(messageBuffer));
			lastmsg = 0
		}
		worldTime = worldTime + 1
		//lastmsg = 0
		//fmt.Println("The time is :",worldTime);
		generateTraderRequests()
		if lastmsg < 0 {
			lastmsg = 0
		} //to check smaller entries than the last
		//successfull one from last time.
		i := lastmsg
		for i < len(messageBuffer) {
			m := messageBuffer[i]
			if m.time > worldTime {
				// TODO(klaus): this is never used, is it intended?
				lastmsg = i
				//fmt.Println("lastMessage is",lastmsg);
				i = len(messageBuffer)
			}

			if m.time == worldTime {
				//fmt.Println("The message is ",m.time,"  ",m.content);
				if m.mtype == "VOTE" {
					processIncomingQNew(m)
					//if (m.sender < 20) {
					//vd[m.receiver].Incoming_Q[m.sender] = append(vd[m.receiver].Incoming_Q[m.sender],m)
					//process_incoming_Q(m.sender,m.receiver);
				} else {
					_ = processMessage(m, m.receiver)
				}
				//processMessage(m,m.receiver);
				if m.mtype == "BlockTrigger" {

					processBlock(m.content)
				}
			}
			i = i + 1
		}
	}
	// Finishing up
	endStatus()
}

func initWendy() {
	fmt.Println("Wendy initializing")
	totalSpread = 0
	totalVotes = 0
	maxSpread = 0
	worldTime = 0
	i := 19
	for i > 0 {
		vd[i].X_Coord = rand.Intn(100)
		vd[i].Y_Coord = rand.Intn(100)
		vd[i].LastDoneTX = -1
		j := 19
		for j > 0 {
			vd[i].OtherSeqNos[j] = -1
			j = j - 1
		}
		i = i - 1
	}
}

func endStatus() {
	//showVotes(1);
	//showTX(1);
	//showTX(2);
	//showTX(3);
	//showTX(4);
	//fmt.Println("Blocked Votes: ",len(vd[1].IncomingQ[2]))
	//fmt.Println(vd[1].IncomingQ[2])
	fmt.Println("Final Statistics: TX: ", totalTX, ", Delayed:", totalDelayedTX, ",Ins. Votes:", delayed_insufficient_votesTX)
}
