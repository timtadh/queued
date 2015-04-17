package net

/* queued
 * Author: Tim Henderson
 * Email: tadh@case.edu
 * Copyright 2013 All Right Reserved
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *
 *  * Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 *  * Neither the name of the queued nor the names of its contributors may be
 *    used to endorse or promote products derived from this software without
 *    specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

import "testing"

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"strconv"
)

import (
	"github.com/timtadh/queued/queue"
)


func init() {
	rand.Seed(int64(binary.LittleEndian.Uint64(rand_bytes(8))))
}

func rand_bytes(length int) []byte {
	if urandom, err := os.Open("/dev/urandom"); err != nil {
		log.Fatal(err)
	} else {
		slice := make([]byte, length)
		if _, err := urandom.Read(slice); err != nil {
			log.Fatal(err)
		}
		urandom.Close()
		return slice
	}
	panic("unreachable")
}

func TestEncodeMessage(t *testing.T) {
	check := func(msg []byte, correct string) {
		if string(msg) != correct {
			t.Errorf("'%v'", string(msg))
		}
	}
	check(EncodePlainMessage("OK", nil), "OK\n")
	check(EncodePlainMessage("WIZARD", []byte("hi")), "WIZARD hi\n")
	check(EncodeB64Message("ERROR", []byte("hi")), "ERROR aGk=\n")
}

func TestDecodeMessage(t *testing.T) {
	check := func(line, c_cmd string, c_msg []byte) {
		cmd, rest := DecodeCmd([]byte(line))
		if c_cmd != cmd {
			t.Errorf("bad cmd, '%v' != '%v'", c_cmd, cmd)
		}
		if c_msg != nil {
			msg, err := DecodeB64(rest)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(c_msg, msg) {
				t.Errorf("bad msg, '%v' != '%v'", c_msg, msg)
			}
		} else if c_msg == nil && rest != nil {
			t.Errorf("bad msg, '%v' != '%v'", c_msg, rest)
		}
	}

	check("ERROR aGk=\n", "ERROR", []byte("hi"))
	check("OK", "OK", nil)
}

func sender_to_reciever(recv <-chan []byte) <-chan byte {
	send := make(chan byte)
	go func() {
		for block := range recv {
			for _, b := range block {
				send <- b
			}
		}
		log.Println("Closed")
		close(send)
	}()
	return send
}

func TestConnection(t *testing.T) {
	server := NewServer(func() Queue { return queue.NewQueue(true) })
	connect := func() (chan<- []byte, <-chan []byte) {
		send := make(chan []byte)
		r := sender_to_reciever(send)
		s := make(chan []byte)
		go server.Connection(s, r).Serve()
		return send, s
	}

	test := func() {
		send, recv := connect()
		l := make([][]byte, 0, 25)
		for i := 0; i < rand.Intn(25)+10; i++ {
			item := rand_bytes(rand.Intn(32) + 2)
			l = append(l, item)
			send <- EncodeB64Message("ENQUE", item)
			cmd, data := DecodeCmd(<-recv)
			if cmd != "OK" && data != nil {
				t.Fatal("Expected an OK response")
			}
			send <- EncodeB64Message("HAS", queue.Hash(item))
			cmd, data = DecodeCmd(<-recv)
			if cmd != "TRUE" && data != nil {
				t.Fatal("Expected an TRUE response")
			}
		}
		for _, item := range l {
			send <- EncodePlainMessage("DEQUE", nil)
			cmd, rest := DecodeCmd(<-recv)
			if cmd != "ITEM" || rest == nil {
				t.Fatal("expected an item")
			}
			q_item, err := DecodeB64(rest)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(q_item, item) {
				t.Fatal("items should have equalled each other")
			}
		}
		send <- EncodePlainMessage("DEQUE", nil)
		cmd, rest := DecodeCmd(<-recv)
		if cmd != "ERROR" {
			t.Fatal("expected an error")
		}
		if rest == nil {
			t.Fatal("expected queue empty message")
		}
		msg, err := DecodeB64(rest)
		if err != nil {
			t.Fatal(err)
		}
		if string(msg) != "queue is empty" {
			t.Fatal("expected queue empty message")
		}
		close(send)
		<-recv
	}
	test()
	test()
	test()
	test()
	test()
}

func TestMultiConnection(t *testing.T) {
	server := NewServer(func() Queue { return queue.NewQueue(true) })
	connect := func() (chan<- []byte, <-chan []byte) {
		send := make(chan []byte)
		r := sender_to_reciever(send)
		s := make(chan []byte)
		go server.Connection(s, r).Serve()
		return send, s
	}

	test := func(final bool) {
		send, recv := connect()
		l := make([][]byte, 0, 25)
		for i := 0; i < rand.Intn(25)+10; i++ {
			item := []byte("same item for all!") // that way it doesn't matter
			// who deques
			l = append(l, item)
			send <- EncodeB64Message("ENQUE", item)
			cmd, rest := DecodeCmd(<-recv)
			if cmd != "OK" && rest != nil {
				t.Fatal("Expected an OK response", cmd, string(rest))
			}
			send <- EncodeB64Message("HAS", queue.Hash(item))
			cmd, data := DecodeCmd(<-recv)
			if cmd != "TRUE" && data != nil {
				t.Fatal("Expected an TRUE response")
			}
		}
		for _, item := range l {
			send <- EncodePlainMessage("DEQUE", nil)
			cmd, rest := DecodeCmd(<-recv)
			if cmd != "ITEM" || rest == nil {
				t.Fatal("expected an item")
			}
			q_item, err := DecodeB64(rest)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(q_item, item) {
				t.Fatal("items should have equalled each other")
			}
		}
		if final {
			size := 1
			for size > 0 {
				send <- EncodePlainMessage("SIZE", nil)
				cmd, sSize := DecodeCmd(<-recv)
				if cmd != "SIZE" {
					t.Fatal("expected a SIZE")
				}
				i, err := strconv.Atoi(string(bytes.TrimSpace(sSize)))
				if err != nil {
					t.Fatal(err)
				}
				size = i
			}
			send <- EncodePlainMessage("DEQUE", nil)
			cmd, rest := DecodeCmd(<-recv)
			if cmd != "ERROR" {
				t.Fatal("expected an error")
			}
			if rest == nil {
				t.Fatal("expected queue empty message")
			}
			msg, err := DecodeB64(rest)
			if err != nil {
				t.Fatal(err)
			}
			if string(msg) != "queue is empty" {
				t.Fatal("expected queue empty message")
			}
		}
		close(send)
		<-recv
	}
	go test(false)
	go test(false)
	go test(false)
	go test(false)
	test(true)
}
