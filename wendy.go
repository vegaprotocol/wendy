package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
)

//******************************************************************
//**
//** TODO List
//**
//**   * Test, test, test
//**   * Verify that I handeled all slices correctly and didnt'
//**     use a reference when I should have copied
//**   * Clean up the data types. Old transactions can be deleted
//**     On the long run, I also need to cap D and clear the message buffer.
//**   * Market Identifiers can be optimised.
//**     They are currently implemented in the blocking function
//**     It would actually even better to have validator (id.mid) to
//**     keep them completely separate. Only on block delivery would all
//**     Q of the leader be flashed. This would massively save resources on
//**     recompute(). Doesn't matter for the simulation, though
//**   * Insert realistic values for network delay and blockchain performance
//**   * Clean up junk code and comberson stuff that is c in go
//**   * The handling of sequence numbers is somewhat sub-optimal. Could
//**    do with a more clever data structure
//**  Issues
//**
//*****************************************************************************
//**
//** Note for a real implementation
//** The sequence number thing is fragile. Some recovery is needed
//**	* If a sequence number relates to a request that got
//**	  scheduled, we can skip it
//**    * Some resend mechanism
//**
//**  There's some checkes we don't do yet; for example, we know that
//**  Validatoes don't vote twice for the same tx, so we don't check it.
//**
//**  We have no signatures or proof-of-following-the-protocol in here.
//**  In the real implementation, every block proposal needs to come
//**  with the necessary signatures proving it's valid. If things get
//**  wired, that might be a lot of signatures; there is probably
//**  a lot of room for improvement there to optimise this, e.g.,
//**  resending every vote I get right away to all others (with signatures,
//**  time and order) so they can replicate my state (or use a reliable broadcast
//**  so they know).
//**  To keep latency down, we could send the signature blob while the
//**  block with the corresponding txs is processed by the blockchain,
//**  and do a post-validity check. New can of worms though...
//**********************************************************************

//******************************************************************
//** Definition of message formats
//******************************************************************

//* Generic Message (send by everyone).
type message struct {
	sender   int
	receiver int
	mtype    string
	time     int // Delivery time; this is actually part of the network
	content  string
}

//* Transaction as send by traders (as content of a message)
type Transaction struct {
	Marketid int    `json:"MID"`
	Payload  string `json:"payl"`
}

type Block struct {
	Payload string `json:"payl"`
}

// Voting message
type Vote struct {
	Seq_Number    int
	Marketid      int    `json:"MID"`
	Payload       string `json:"payl"`
	Received_time int
	Sender        int
}

//*********************************************************************
//**
//** Validator State
//**
//*********************************************************************
// Transactions as saved in the internal state of a validator
type Transaction_v struct {
	Marketid      int    `json:"MID"`
	Payload       string `json:"payl"`
	Received_time int
	Seq_Number    int
	blockers      []int  // Which transaction (by number) is blocking me
	Votes         []Vote // Who voted for this transaction
}

type validator_state struct {
	Id              int
	Sequence_number int
	Other_Seq_Nos   [20]int // Last sequence number seen by others
	Transactions    []Transaction_v
	Incomming_Q     [20][]message // Votes we can't process yet due to the wrong sequence number
	Br              [][]string    // Used to compute the blockings. For each transactions, store the blocking transactions
	Q               []string      // List of requests ready for the next block
	D               []string      // List of requests already dealt with
	U               []string      // List of all known requests not in Q or D

}

var worldtime int
var messagebuffer []message
var vd [20]validator_state // Max numvber of validators for now
// Debugging and controlling
var n int // The first validator is ID 1, not 0
var t int
var r int
var debug_leader int // The party we watch for debug messages
var blockchain_delay int
var blockchain_rnd int
var msg_delay int
var msg_rnd int
var delay int   // How many ticks to wait until a trader sends a new message
var runtime int // How long do we want the simulator to run ?
//Statistical Stuff
var totalspread int
var totalvotes int
var max_spread int

//*********************************************************************
//*
//* Tools
//*
//**********************************************************************
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
func Checkpoint(s string, code int) {
	// Debugging output
	if code == 7 {
		fmt.Println("Checkpoint ", s)
	}
}

//*********************************************************************
//*
//* Network Simulation
//* Essentially, the network is an array of strings; tjhe index
//*  indicats at whot global time the string is arriving at its
//*  destination.
//* For now we're a bit wasteful, as we don't remove historic
//* entries
//* As for IDs, we reserve 1-100 for validatorsm and 100+ for traders
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
func send_message(payload string, mtype string, sender int, receiver int) {
	time := worldtime + msg_delay // Deterministic version that assures no message
	// is out of order.
	if msg_rnd > 0 {
		time += rand.Intn(msg_rnd)
	}
	if sender == r {
		fmt.Println("3 Sends ", payload, "to ", receiver, ". Will arrive at ", time)
	}
	if sender == 999999 {
		fmt.Println("Sending Message: ", mtype, payload, time, "from", sender, " to ", receiver)
	}
	// Messages to self take no time
	if sender == receiver {
		time = worldtime + 1
	}

	send_message_withtime(payload, mtype, sender, receiver, time)
}

func insert_sorted(m message) {
	messagebuffer = append(messagebuffer, m)
	blen := len(messagebuffer) - 1
	if blen >= 0 {
		i := 0
		for m.time > messagebuffer[i].time && i <= blen+1 {
			i++
		} // for
		copy(messagebuffer[i+1:], messagebuffer[i:])
		messagebuffer[i] = m
	}
	//		iii:=0;
	//		fmt.Println("The SORTED message list after inserting time",m.time," is:")
	//		for (iii<len(messagebuffer)) {
	//			fmt.Print(messagebuffer[iii].time," ");
	//		iii++
	//		}
	//		fmt.Println(" ");

}

func send_message_withtime(payload string, mtype string, sender int, receiver int, time int) {
	m1 := message{}
	m1.sender = sender
	m1.time = time
	m1.receiver = receiver
	m1.content = payload
	m1.mtype = mtype
	//messagebuffer=append(messagebuffer,m1);
	insert_sorted(m1)
}

func multicast_message_withtime(payload string, mtype string, sender int, time int) {
	// Note: It is important to send a message to self as well for the
	// counting arguments to work!
	i := 0
	for i < n {
		send_message_withtime(payload, mtype, sender, i+1, time)
		i = i + 1
	}
}

func multicast_message(payload string, mtype string, sender int) {
	// Note: It is important to send a message to self as well for the
	// counting arguments to work!
	i := 0
	if sender == r {
		fmt.Println("3 Multicases ", payload)
	}
	for i < n {
		send_message(payload, mtype, sender, i+1)
		i = i + 1
	}
}

func generate_trader_requests() {
	// This function is called any tick.
	// We can send a trade per tick, or more or less
	t1 := &Transaction{Marketid: 5, Payload: "Test"}
	sender := rand.Intn(1000) + 100 // IDs 1-100 are reserved for valifdators
	//t1.Marketid = rand.Intn(5);
	t1.Marketid = 1
	temp, _ := json.Marshal(t1.Marketid + 100*worldtime)
	t1.Payload = string(temp)
	tenc, _ := json.Marshal(t1)
	//fmt.Println("Error is ",err);
	//fmt.Println("Length: ", len(tenc));
	//var t2 Transaction;
	//json.Unmarshal(tenc,&t2);
	// fmt.Println("Reconverted: ",t2.payload,t2.marketid)
	//fmt.Println("Encoded ",t1.payload,t1.marketid," ",string(tenc));
	//fmt.Println("Encoded ",string(tenc));
	payload := string(tenc)
	if (worldtime/delay)*delay == worldtime { // Slow things down a little for testing
		multicast_message(payload, "TX", sender)
	}
}

func process_block(b string) { // Simulate the underlying blockchain, i.e.,
	// Once in a while pick up a block
	// We do this using the message interface

	Checkpoint("A block has been finished", 2)
	var leader = 1
	if n > 1 {
		leader = rand.Intn(n-1) + 1
	}
	leader = 1 // For testing purposes, fix the leder. TODO: Delete this line.

	// We simulate the blockchain via the network: essentially,
	// the leader sends a message with the block content to
	// all parties, and this scheduled the block (this message
	// is on it's way longer than normal ones as it simulates
	// the whole back and forth of the blockchain
	block_delay := blockchain_delay
	if blockchain_rnd > 0 {
		block_delay += rand.Intn(blockchain_rnd)
	}
	var q2, _ = json.Marshal(vd[leader].Q)
	fmt.Println("Proposing new block: ", len(vd[leader].Q), string(q2))
	fmt.Println("Number of TXs in block: ", len(vd[leader].Q))
	fmt.Println("Number of Txs delayed by Wendy: ", len(vd[leader].U))
	fmt.Println("Out of order votes in leader buffer: ", len(vd[leader].Incomming_Q))
	fmt.Println("Worldtime is :", worldtime)
	tmp := 1
	for tmp <= n {
		fmt.Println("Seq number of ", tmp, "is ", vd[leader].Other_Seq_Nos[tmp])
		tmp = tmp + 1
	}
	//fmt.Println("Leader Buffer: ",vd[leader].Incomming_Q);

	multicast_message_withtime(string(q2), "Block", leader, worldtime+block_delay)

	// This is a message we just send to the network so that
	// this function gets triggered again when the next block
	// can be processed. For Tendermint, this happens right after
	// the current block has arived. For concurrent blockchains
	// this could be faster than the multicast.
	send_message_withtime(string(q2), "BlockTrigger", 0, 0, worldtime+block_delay+1)
}

//*************************************************************************************
//**
//** Actual Wendy Implementation
//**
//*************************************************************************************\

func is_blocked(m message, id int) bool {
	// If blocked means that it is possible that a transaction we haven't
	// even seen yet might require preference over the transaction in m.
	// This means we cannot process that request yet
	// So far we only implement the first scheme, where a message is blocked
	// if it has received less than t+1 votes.
	current_index := id_by_payload(m.content, id)
	if len(vd[id].Transactions[current_index].Votes) > t {
		fmt.Println("Message ", m.content, " is un blockedi wirh ", len(vd[id].Transactions[current_index].Votes), "votes.")
		return (false)
	}
	return true

}
func is_blocked_t(s string, id int) bool {
	// If blocked means that it is possible that a transaction we haven't
	// even seen yet might require preference over the transaction in m.
	// This means we cannot process that request yet, or all that need to be
	// in the same block as it.
	var temp bool
	temp = true
	current_index := id_by_payload(s, id)
	//fmt.Println(vd[id].Transactions[current_index].Votes,s,id)
	//fmt.Println(len(vd[id].Transactions[current_index].Votes),s,id)
	if len(vd[id].Transactions[current_index].Votes) > t {
		//return (false);
		temp = false
	}
	if id == debug_leader && temp == true {
		fmt.Println(s, " is blocked")
	}
	return (temp)
}

func seen_earlier(s1 string, s2 string, id1 int, id int) bool {
	// From the point of view of id, has seen id seen s1 before s2 ?
	// If they haven't been seen, we force in a false
	// If s1 has been seen and s2 has not, we need to return true
	index1 := id_by_payload(s1, id)
	index2 := id_by_payload(s2, id) //TODO: Can that be -1 ?
	seq1 := -1
	seq2 := -1
	i := len(vd[id].Transactions[index1].Votes) - 1
	for i >= 0 {
		if vd[id].Transactions[index1].Votes[i].Sender == id1 {
			seq1 = vd[id].Transactions[index1].Votes[i].Seq_Number
		}
		i--
	}
	i = len(vd[id].Transactions[index2].Votes) - 1
	for i >= 0 {
		if vd[id].Transactions[index2].Votes[i].Sender == id1 {
			seq2 = vd[id].Transactions[index2].Votes[i].Seq_Number
		}
		i--
	}
	//fmt.Println("Sequence Numbers: ",seq1,seq2);
	if seq2 == -1 {
		return (true)
	} // If s2 hasn't been seen yet, s1 had been seen earlier
	if seq1 == -1 {
		return (false)
	} // If s2 hasb't been seen either, it hasn't
	// This means that if neither has been seem, we return false;
	//   If that is the case, how did we get here ?
	if seq1 < seq2 {
		return (true)
	} // If both have been seen, their sequence number decides
	return (false)
}

func is_blocking(s1 string, s2 string, id int) bool {
	// Is s1 blocking s2 (i.e., it needs to be in the same or an earlier block ?
	//	s1 is blocking s2 if it is possible that all honest parties saw s1
	//	before s2. Thus, if >=t+1 parties saw s2 before s1, s1 is not blocking s2,

	//so, we go through all ids and count seen_earlier(s2,s1...). If that id
	// hasn't seen s1 at all, it'll report false

	// If the two transactions have a different marketid,
	// then they're not blocking each other.
	if vd[id].Transactions[id_by_payload(s1, id)].Marketid != vd[id].Transactions[id_by_payload(s2, id)].Marketid {
		return false
	}

	i := n
	counter := 0
	for i >= 1 {
		if seen_earlier(s2, s1, i, id) {
			//fmt.Println(i," has seen ",s2," before ",s1);
			counter++
		}
		i--
	}
	// Counter now says how oftern s2 has been seen before s1.
	// If that's t+1 or more, we return false
	// BUG: This seems to return true too much
	//return(false);
	if id == debug_leader && counter < t+1 {
		fmt.Println(s1, " is blocking ", s2, " because ", counter)
	}

	return (counter < t+1)
}

func is_delivered(payload string, id int) bool {
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

func index_by_string(array []string, query string) int {
	i := len(array) - 1
	for i >= 0 {
		if array[i] == query {
			return i
		}
		i--
	}
	return -1
}

func id_by_payload(payload string, id int) int {
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
	//************************************************************************************
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
	//***************************************************************************************
	var finished bool
	var index int

	finished = false
	for !finished {
		finished = true
		// Whenever the set of unprocessed messages changes,
		// Recompute all blocking sets B_r for all known and
		// unprocessed r
		// All unprocessed messages are in U.
		i := len(vd[id].U) - 1
		j := len(vd[id].U) - 1

		// Now there has to be a much better way to do this; I need Br and
		// to_be_moved to have as many entries as U
		// Can I do to_be_moved[len(vd[id].U)] ??
		vd[id].Br = [][]string{}
		var to_be_moved []bool
		to_be_moved = []bool{}

		ii := len(vd[id].U) - 1
		for ii >= 0 {
			vd[id].Br = append(vd[id].Br, []string{}) // Create an entry for every entry in U
			to_be_moved = append(to_be_moved, false)
			ii--
		}
		//if (id==debug_leader) {
		//	fmt.Println("Overview 1:");
		//	fmt.Println("------------------------------------");
		//	fmt.Println(vd[id].Br);
		//	fmt.Println("------------------------------------");
		//}
		for i >= 0 {
			vd[id].Br[i] = []string{vd[id].U[i]} // everyone is in their own blocking set
			j = len(vd[id].U) - 1
			for j >= 0 {
				if i != j && is_blocking(vd[id].U[j], vd[id].U[i], id) { // Don't add me to my blocking set twice
					vd[id].Br[i] = append(vd[id].Br[i], vd[id].U[j])
				}
				j--
			} // for j
			i--
		} // for i
		if id == debug_leader {
			fmt.Println("Overview Br:")
			fmt.Println("------------------------------------")
			fmt.Println(vd[id].Br)
			fmt.Println("------------------------------------")
		}
		// Now, for every set Br, if any payload in Br is blocked, so is r
		// So we loop through all Brs, then through all payloads, and then
		// see if those are blocked.
		// If not, we move them from U to Q
		// We also need to move all entries in Br
		//fmt.Println("U: ", len(vd[id].U), "Q: ",len(vd[id].Q), " REC", len(vd[id].Br));

		var blocked bool
		i = len(vd[id].Br) - 1
		for i >= 0 {
			blocked = false
			j = len(vd[id].Br[i]) - 1
			for j >= 0 {
				if is_blocked_t(vd[id].Br[i][j], id) {
					blocked = true
					if id == debug_leader {
						fmt.Println("Blockage for  ", i, " because ", j, vd[id].Br[i][j])
					}
				} //if
				j = j - 1
			} // for
			if !blocked {
				Checkpoint("What's going oooooooooooooooon", 3)

				k := len(vd[id].Br[i]) - 1
				for k >= 0 {
					// Now I need to find the index back from the string *SIGH*
					Checkpoint("I like to move it move it", 3)
					index = index_by_string(vd[id].U, vd[id].Br[i][k])
					if index > -1 {
						to_be_moved[index] = true
						Checkpoint("I really like to move it", 3)
					}
					k = k - 1
				} //for
			} // if (!blocked)
			i = i - 1
		} //for
		i = len(vd[id].U) - 1
		for i >= 0 {
			if to_be_moved[i] {
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

//**********************************************************************************
//** One of Wendy's core functions: Get in a message and react on it.
//** We also use this to monitor the blockchain
//** What this function does is
//**   * If we see a transaction first, store its data issue a vote
//**   *  If the market_identifier of that tx is 0, it is pushed right to the blockchain
//**      and not touched anymore.
//**   * If we see a vote, count it
//**   * If we see a block, remove all realated messages from the corresponding Queues
//**   * If a vote comes in, call recompute to rebuild the internal data structures
//**     and decide if a tx is through.
//**
//**  returns false if the message vouldn't be processed yet (i.e., it is a vote
//**  with a too high sequence number/.
//*********************************************************************************
func process_message(m message, id int) bool {
	// Function for Validator id to evaluate the incomming message m
	// This is the core of Wendy

	if id == debug_leader {
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
			time := int(worldtime - (value / 100))
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

	if m.mtype == "Block" { // The underlying blockchain finished some
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
		//q2 := []string{};

		var q2 []string
		json.Unmarshal([]byte(m.content), &q2)
		i := len(q2) - 1
		for i >= 0 {
			// Append the transaction to our list of finished transactions
			vd[id].D = append(vd[id].D, q2[i])
			//Now we need to delete it from all our buffers
			// And then recompute the blockings (Uff)
			// For now, we only delete it from Q and U
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
		// TODO: We might want to clean up some of our datastructures (e.g., Transactions)
		// Eventually
	}

	if m.mtype == "TX" { // We got a new transaction
		// If the market identifier is 0, push
		// the transaction straight into Q. Or better,
		// maintain a second set for that, as we don't want
		// those transactions to cause a recompute
		// (Not that it matters here, but it matters for
		// the final computation. Actually, for the real thing,
		// we probaby want to redo the recompute thing
		// so that it only recomputes the affected market identifier.
		TX := Transaction_v{}
		TX.Received_time = worldtime
		TX.Seq_Number = vd[id].Sequence_number
		t2 := Transaction{}
		json.Unmarshal([]byte(m.content), &t2)
		TX.Payload = t2.Payload
		//fmt.Println("Incomming Vote for validator",id, ": ", TX.Payload);
		TX.Marketid = t2.Marketid
		vote := Vote{}
		vote.Received_time = TX.Received_time
		vote.Seq_Number = TX.Seq_Number
		vote.Payload = TX.Payload
		vote.Marketid = TX.Marketid
		vote.Sender = id
		var m2, _ = json.Marshal(vote)
		//TX.blockers :=int {};
		//TX.voters :=int{};
		//vd[id] is the state of the current validator
		if !seen(TX.Payload, id) {
			//fmt.Println("New Message");
			vd[id].Sequence_number++
			vd[id].Transactions = append(vd[id].Transactions, TX)

			//** If the MarketID is 0, that means the message needs no fairness
			//** Thus, on seeing the transaction, we put it right into our output
			//** Queue. For this simulation, we only do this when we receive the
			//** Transaction, not on a vote (i.e., if a trader sends a message to only
			//** a few traders, or has a slow connection to those traders, suchs to
			//** be them.
			//** We also don't vote on this message anymore, or do any checks for
			//** dublicates here.
			if vote.Marketid == 0 {
				//fmt.Println("Appending ",TX.Payload," to leader ", id);
				vd[id].Q = append(vd[id].Q, TX.Payload)
			}
			if vote.Marketid != 0 {
				//fmt.Println("Appending ",TX.Payload," to unprocessed queue ", id);
				vd[id].U = append(vd[id].U, TX.Payload)
				if id == r {
					fmt.Println("# Saw TX, votes ", vote.Seq_Number)
				}
				multicast_message(string(m2), "VOTE", id)
			}
			//fmt.Println("Queue is now ",len(vd[id].Q));

		} // seen
	} // TX

	if m.mtype == "VOTE" {
		// TODO: If the seqnos of the vote don't fit, we need to hold it back.
		vote := Vote{}
		TX := Transaction_v{}
		json.Unmarshal([]byte(m.content), &vote)
		if 10 == debug_leader {
			totalspread = totalspread + vote.Seq_Number - vd[id].Other_Seq_Nos[m.sender]
			totalvotes++
			if vote.Seq_Number-vd[id].Other_Seq_Nos[m.sender] > max_spread {
				max_spread = vote.Seq_Number - vd[id].Other_Seq_Nos[m.sender]
			}

			fmt.Println("Receive Vote seq.", vote.Seq_Number, " from P", m.sender, ". Was expecting ", vd[id].Other_Seq_Nos[m.sender]+1)
			fmt.Println("Avg Spread: ", totalspread/totalvotes, "(", max_spread, ")")
		}

		//if (m.sender ==3 && m.receiver ==1) {
		if m.sender == r {
			fmt.Println("3 Voted with no", vote.Seq_Number)
			fmt.Println("Expected ", vd[id].Other_Seq_Nos[m.sender]+1)
		}
		if vote.Seq_Number != 0 && vote.Seq_Number != vd[id].Other_Seq_Nos[m.sender]+1 {
			//vd[id].delayed_votes = append(vd[id].delayed_votes,m);
			//delayed_votes = append(delayed_votes,m);
			return false
			//We're cheap for now, and just resend (Will screw up timing, s fix TODO)
			//send_message_withtime(m.content,m.mtype,m.sender, m.receiver ,worldtime+20)

			//fmt.Println("Message out of order from",m.sender,". Expecting ",vd[id].Other_Seq_Nos[m.sender]+1," got ",vote.Seq_Number);
		} else {

			TX.Payload = vote.Payload
			vd[id].Other_Seq_Nos[m.sender] = vote.Seq_Number
			if !seen(vote.Payload, id) {
				vote.Seq_Number = vd[id].Sequence_number
				vote.Sender = m.sender
				vd[id].Sequence_number++
				TX.Received_time = worldtime
				TX.Seq_Number = vote.Seq_Number
				//TX.voters=append(TX.voters,m.sender);
				vd[id].Transactions = append(vd[id].Transactions, TX)
				vote.Received_time = worldtime
				var m2, _ = json.Marshal(TX)
				multicast_message(string(m2), "VOTE", id)
			} //seen
			// Manage votes
			// TODO: Might need review if I count votes double.
			// Essentially, we need to
			//  -- keep a vote counter & list
			//    --> If vote not in the list, add it
			//    --> Counter = size of list
			//  -- recompute if we get a new vote in
			//  -- Manage sequence numbers in an intelligent way.
			//    --> If another validator skips a sequence number,
			//    --> The message with that number goes onto a heap (linked to that sender).
			//    --> Once we get the next following sequence number for that sender,
			//    		--> We throw then entire heap into the network again
			//    		--> Or, we process the entire heap recursively
			//              --> Careful not to create an endless loop here, needs some thinking
			//		--> Don't forget each vote needs its own time!

			current_index := id_by_payload(TX.Payload, id)
			if current_index > -1 {
				//vd[id].Transactions[current_index].voters = append(vd[id].Transactions[current_index].voters,m.sender);
				vd[id].Transactions[current_index].Votes = append(vd[id].Transactions[current_index].Votes, vote)
				if id == debug_leader {
					fmt.Println(id, " saw ", len(vd[id].Transactions[current_index].Votes), "votes for ", TX.Payload)
					fmt.Println("Sequence numbers are:")
					for iii := 0; iii < len(vd[id].Transactions[current_index].Votes); iii++ {
						fmt.Println(vd[id].Transactions[current_index].Votes[iii].Sender, vd[id].Transactions[current_index].Votes[iii].Seq_Number)
					} //for
					fmt.Println("U: ", len(vd[id].U), "Q: ", len(vd[id].Q), " REC", len(vd[id].Br))
				} //if
			}

			recompute(id)

			//TODO: Eliminate double votes. Not needed for the statistics,
			// but sorta vital for the real thing

			//if (! is_delivered(TX.Payload,id)) {
			//		fmt.Println("TODO");
			//}

			// Processing:
			// IF we have a request T such that T is not in D, an Br(T) is empty,
			//   and votes(T) > t, then Br(T) = T.

			// For all T for which Br(T) is non-empty.
			//   for all known T2 not in Br(T)
			//     if votes (T2) >0
			//     if blocks(T2,T)
			//        add T2 to Br(T)
			//   for all T2 in Br(T),
			//     if (!blocks (T2,T)
			//        remove T2 from Br(T)

			// forall known and undelivered T
			//   if no request in T is blocked
			//       add all of Br to Q

			//TODO Verify: Can I do the whole sequence number thing differently
			//     in is_blocked and block every tx linked to missing
			//     sequence numbers ?????

		} // else (Vote has been processed)
	} //VOTE
	return true
}

func process_incomming_Q(sender int, id int) {
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
	//	Incomming_Q[][], use go sort, and convert it back. Which is ugly. Look for a
	//	better solution before doing that :)
	i := len(vd[id].Incomming_Q[sender]) - 1
	qlen := i
	if process_message(vd[id].Incomming_Q[sender][i], id) {
		vd[id].Incomming_Q[sender] = RemoveIndexMsg(vd[id].Incomming_Q[sender], i)
		j := i - 1
		for j < qlen {
			qlen = len(vd[id].Incomming_Q[sender]) - 1
			for j >= 0 {
				if process_message(vd[id].Incomming_Q[sender][j], id) {
					vd[id].Incomming_Q[sender] = RemoveIndexMsg(vd[id].Incomming_Q[sender], j)
				} // if
				j--
			} // for
			j = len(vd[id].Incomming_Q[sender]) - 1
		}
	} // if
}

func seq(m message) int {
	var v Vote
	if m.mtype != "VOTE" {
		return -1
	}
	_ = json.Unmarshal([]byte(m.content), &v)
	return v.Seq_Number

}

func process_incomming_Q_new(m message) {
	// The i queue contains all messages id received from sender that could not
	// be processed for now because they are voting messages with a future sequence number.
	// The last message here is that last added; if this one is out of order, all others
	// in the queue stay so. Otherwise, we replay the entire queue until it didn't
	//iii:=0
	id := m.receiver
	qlen := len(vd[id].Incomming_Q[m.sender]) - 1
	qlen2 := len(vd[id].Incomming_Q[m.sender]) - 1
	// First, we process the new message..
	// if it goes through, we can eat up other messages in the queue
	// and don't need to sort.
	// Else, it is no part of the queue and we need to resort.
	if process_message(m, m.receiver) {
		//iii=0;
		//fmt.Println("The INITIAL list for V",id," and sender ",m.sender,"is")
		//for (iii<len(vd[id].Incomming_Q[m.sender])) {
		//	fmt.Print(seq(vd[id].Incomming_Q[m.sender][iii])," ");
		//	iii++
		//}
		//fmt.Println(" ");
		j := 0
		for j <= qlen2 {
			if process_message(vd[id].Incomming_Q[m.sender][j], id) {
				j++
				//fmt.Println("Processing Seq:",seq(vd[id].Incomming_Q[m.sender][j-1]));
			} else {
				qlen2 = -1
			}
			//fmt.Println("Before", len(vd[id].Incomming_Q[m.sender]));
			//fmt.Println("J was",j);
		} //for j<qlen2
		// Now we have processed a number of transactions, and we need to
		// delete them from the message list
		//		iii=0;
		//		fmt.Println("The BEFORE list for V",id," and sender ",m.sender,"is")
		//		for (iii<len(vd[id].Incomming_Q[m.sender])) {
		//		fmt.Print(seq(vd[id].Incomming_Q[m.sender][iii])," ");
		//		iii++
		//		}//for
		//		fmt.Println("J is ",j, "and the seq no in question was",seq(m));
		if j > 0 {
			vd[id].Incomming_Q[m.sender] = append(vd[id].Incomming_Q[m.sender][:0], vd[id].Incomming_Q[m.sender][j:]...)
		}
		//		iii=0;
		//		fmt.Println("The AFTER list for V",id," and sender ",m.sender,"is")
		//		for (iii<len(vd[id].Incomming_Q[m.sender])) {
		//		fmt.Print(seq(vd[id].Incomming_Q[m.sender][iii])," ");
		//		iii++
		//		}//for
		//		fmt.Println(" ");

		// If we didn't process the new element successfully, we need to insert
		// it at its right position in the incomming queue. No other element
		// in that queue needs processing, as none of them can have gotten
		// unblocked.
	} else {
		//fmt.Println("No processing. Adding ",seq(m));
		vd[id].Incomming_Q[m.sender] = append(vd[id].Incomming_Q[m.sender], m)
		if qlen >= 0 {
			i := 0
			for seq(m) > seq(vd[id].Incomming_Q[m.sender][i]) && i <= qlen+1 {
				i++
			} // for
			//fmt.Println("Inserting at place ",i,qlen);
			//fmt.Println(vd[id].Incomming_Q[m.sender]);
			copy(vd[id].Incomming_Q[m.sender][i+1:], vd[id].Incomming_Q[m.sender][i:])
			vd[id].Incomming_Q[m.sender][i] = m
			//fmt.Println("The sorted list is");
			//fmt.Println(vd[id].Incomming_Q[m.sender]);
			//iii=0;
			//fmt.Println("The SORTED list for V",id," and sender ",m.sender, "after inserting ",seq(m),"is")
			//for (iii<len(vd[id].Incomming_Q[m.sender])) {
			//	fmt.Print(seq(vd[id].Incomming_Q[m.sender][iii])," ");
			//	iii++
			//}
			//fmt.Println(" ");

		}
	}
}

//********************************************************************************
//**
//** Wendy stops here.
//**
//********************************************************************************

// WIP: CLean the buffers of messages that have already been processed
// Incompatible with lastmsg
// Things that also could be cleaned: TX Buffer
func clean_memory() {
	i := 0
	for i < len(messagebuffer)-1 && messagebuffer[i].time < worldtime {
		i++
	}
	if i > 0 {
		messagebuffer = append(messagebuffer[:0], messagebuffer[i:]...)
	}
}

func network() {
	// Network simulator.
	// Essentially, we just add one tick and see if there's an undelivered
	// for that time, and then process it. There is a special message type
	// BlockTrigger used to simulate the underlying blockchains (i.e., trigger
	// the processing of a new block).
	// TODO: Just like the Incomming_Q, I could sort the messagebuffer.
	// Not sure yet if that's worth the effort though.
	for worldtime < runtime || runtime == 0 {
		worldtime = worldtime + 1
		lastmsg := 0
		lastmsg2 := 0
		fmt.Println("The time is :", worldtime)
		generate_trader_requests()
		i := len(messagebuffer) - 1 //TODO: Needs optimizing :)
		lastmsg = lastmsg2          //Since e're counting down, no need
		if lastmsg < 0 {
			lastmsg = 0
		} //to check smaller entries than the last
		//successfull one from last time.
		for i >= lastmsg {
			m := messagebuffer[i]
			if m.time > worldtime {
				lastmsg2 = i
			} // lastmsg2 now should be the smallest i with a message still to be delivered
			// so everything in the messagebuffer smaller than lastmsg2 can be ignored from now
			// on. For a more serious implementation, we should use some form of ringbuffer
			// here, but for our purposes this will do.

			if m.time == worldtime {
				//fmt.Println("The message is ",m.time,"  ",m.content);
				if m.mtype == "VOTE" {
					process_incomming_Q_new(m)
					//if (m.sender < 20) {
					//vd[m.receiver].Incomming_Q[m.sender] = append(vd[m.receiver].Incomming_Q[m.sender],m)
					//process_incomming_Q(m.sender,m.receiver);
				} else {
					_ = process_message(m, m.receiver)
				}
				//process_message(m,m.receiver);
				if m.mtype == "BlockTrigger" {
					process_block(m.content)
				}
			}
			i = i - 1
		}
	}
}
func network_new() {
	// Network simulator.
	// Essentially, we just add one tick and see if there's an undelivered
	// for that time, and then process it. There is a special message type
	// BlockTrigger used to simulate the underlying blockchains (i.e., trigger
	// the processing of a new block).
	// TODO: Just like the Incomming_Q, I could sort the messagebuffer.
	// Not sure yet if that's worth the effort though.
	for worldtime < runtime || runtime == 0 {
		worldtime = worldtime + 1
		lastmsg := 0
		//fmt.Println("The time is :",worldtime);
		generate_trader_requests()
		if lastmsg < 0 {
			lastmsg = 0
		} //to check smaller entries than the last
		//successfull one from last time.
		i := lastmsg
		for i < len(messagebuffer) {
			m := messagebuffer[i]
			if m.time > worldtime {
				lastmsg = i
				i = len(messagebuffer)
			} // lastmsg2 now should be the smallest i with a message still to be delivered
			// so everything in the messagebuffer smaller than lastmsg2 can be ignored from now
			// on. For a more serious implementation, we should use some form of ringbuffer
			// here, but for our purposes this will do.

			if m.time == worldtime {
				//fmt.Println("The message is ",m.time,"  ",m.content);
				if m.mtype == "VOTE" {
					process_incomming_Q_new(m)
					//if (m.sender < 20) {
					//vd[m.receiver].Incomming_Q[m.sender] = append(vd[m.receiver].Incomming_Q[m.sender],m)
					//process_incomming_Q(m.sender,m.receiver);
				} else {
					_ = process_message(m, m.receiver)
				}
				//process_message(m,m.receiver);
				if m.mtype == "BlockTrigger" {
					process_block(m.content)
				}
			}
			i = i + 1
		}
	}
	// Finishing up
	//fmt.Println(vd[1].Transactions[0].Votes);
	//fmt.Println(vd[1].Transactions[1].Votes);
	//fmt.Println(vd[1].Transactions[2].Votes);
	//fmt.Println(vd[1].Transactions[3].Votes);
	//fmt.Println(vd[1].Transactions[4].Votes);
	//fmt.Println(vd[1].Transactions[5].Votes);
}

func init_wendy() {
	fmt.Println("Wendy initializing")
	totalspread = 0
	totalvotes = 0
	max_spread = 0
	worldtime = 0
	i := 19
	for i > 0 {
		j := 19
		for j > 0 {
			vd[i].Other_Seq_Nos[j] = -1
			j = j - 1
		}
		i = i - 1
	}
}
