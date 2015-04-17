// This package implements a TCP network service which allows clients to queue
// and deque data. The client has the following verbs
//
//  - ENQUE
//  - DEQUE
//  - HAS
//  - SIZE
//  - USE
//
// the server can send the following reponse status words
//
//  - OK
//  - ERROR
//  - ITEM
//  - TRUE
//  - FALSE
//  - SIZE
//
// All messages have the following format:
//
//     COMMAND DATA
//
// COMMAND is a single ASCII word with no spaces. DATA is the rest of the line.
// DATA is optional. For ERROR, ENQUE and ITEM it is Base64 encoded. For SIZE
// reponse it is ASCII.
//
// The client can send any command at any time. The server may at any command
// respond with ERROR if there was a problem processing the command.
//
// USE name
//
//     Use the named queue. You never have to issue this command. If you do not
//     you will automatically be using a queue named "default". If the queue does
//     not this command will create one. The server should respond:
//
//          OK
//
//     name should have no spaces and should be utf8.
//
// ENQUE XXXXXXXXXXXXXXXX
//
//     XXXXXXXXXXXXXXXXXX should be base64 encoded data. If it is not the server
//     will respond with and ERROR.
//
//     Otherwise the server will repond
//
//          OK
//
//     The server should always respond with OK if the command is properly
//     formated. If it does not that means there is an internal error in the
//     server.
//
// DEQUE
//
//     If the queue is empty the server will respond with:
//
//         ERROR cXVldWUgaXMgZW1wdHk=
//
//     Which decodes too:
//
//         ERROR queue is empty
//
//     Otherwise it repondes with
//
//         ITEM XXXXXXXXXXXXXXX
//
// HAS XXXXXXXXXXXXXXXXXXXXXXXXXXX
//
//      HAS checks for the existence of an item. It doesn't send the item but
//      rather the base64 encoded sha265sum. Example for producing this (in
//      Python for simplicity)
//
//         >>> import hashlib
//         >>> hashlib.sha256('item').digest().encode('base64').strip()
//         'SjPqzV+mXysuKHHNExKGtTxBWxMWZtcRc7tuP+WTYbM='
//
//      The client would send
//
//          HAS SjPqzV+mXysuKHHNExKGtTxBWxMWZtcRc7tuP+WTYbM=
//
//      The server would reponse either
//
//          TRUE
//
//      or
//
//          FALSE
//
//
// SIZE
//
//      The server response with the size of the queue encoded as a base10 ascii
//      number. eg.
//
//          SIZE 9231
//
//      For a queue that is 9,231 items long.
//
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

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	logpkg "log"
	"net"
	"os"
	"strings"
)

import (
	netutils "github.com/timtadh/netutils"
)

var log *logpkg.Logger

func init() {
	log = logpkg.New(os.Stderr, "queued/net> ", logpkg.Ltime|logpkg.Lshortfile)
}

type Server struct {
	ln    *net.TCPListener
	newQueue func() Queue
	queues map[string]Queue
}

func NewServer(creator func() Queue) *Server {
	s := &Server{
		newQueue: creator,
		queues: make(map[string]Queue),
	}
	s.queues["default"] = s.newQueue()
	return s
}

/*
Starts a server. This is a blocking call it will run until the server shuts
down. If you want to run this in a seperate thread simply call it in its own
goroutine. Additionally under certain conditions this function may panic.

These include:
    1. There is already a link established.
    2. The server is unable to bind to the port.

The expectation is for these errors to either cause a hard crash or be caught
logged and then crashed.  */
func (self *Server) Start(port int) {
	if self.ln != nil {
		panic("Server already started")
	}
	ln, err := net.ListenTCP(
		"tcp",
		&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: port},
	)
	if err != nil {
		panic(err)
	}
	self.ln = ln
	self.listen()
}

/*
Stop a started server. If there is some problem stopping the server an error
will be returned.  */
func (self *Server) Stop() error {
	if self.ln == nil {
		return fmt.Errorf("Can't close non-existent link")
	}
	return self.ln.Close()
}

func (self *Server) listen() {
	errors := ErrorHandler()
	var EOF bool
	for !EOF {
		con, err := self.ln.AcceptTCP()
		if netutils.IsEOF(err) {
			EOF = true
		} else if err != nil {
			log.Panic(err)
		} else {
			send := netutils.TCPWriter(con, errors)
			recv := netutils.TCPReader(con, errors)
			go self.Connection(send, recv).Serve()
		}
	}
}

/*
Decode messages to/from the server. The server reads and sends line oriented
commands with message bodies base64 encoded. This function handles decoding
those lines for you.  */
func DecodeB64(msg []byte) ([]byte, error) {
	b64 := base64.StdEncoding
	encoded := bytes.TrimSpace(msg)
	rest := make([]byte, b64.DecodedLen(len(encoded)))
	if n, err := b64.Decode(rest, encoded); err != nil {
		return nil, err
	} else {
		rest = rest[:n]
	}
	return rest, nil
}

func DecodeCmd(line []byte) (string, []byte) {
	split := bytes.SplitN(line, []byte(" "), 2)
	command := string(bytes.TrimSpace(split[0]))
	if len(split) > 1 {
		return command, split[1]
	} else {
		return command, nil
	}
}

/*
Encode a message for the server. Handles base64ing the msg. It is expected that
the paramter `cmd` will not have a space in it. If it does problems will occur
on the reciever side.  */
func EncodeMessage(cmd string, msg []byte, enc Encoder) []byte {
	if msg == nil {
		return []byte(cmd + "\n")
	}
	bcmd := []byte(cmd)
	cmdLen := len(bcmd)
	msgEncLen := cmdLen + enc.EncodedLen(len(msg)) + 2
	msgEnc := make([]byte, msgEncLen)
	copy(msgEnc[:cmdLen], bcmd)
	msgEnc[cmdLen] = ' '
	enc.Encode(msgEnc[cmdLen+1:msgEncLen-1], msg)
	msgEnc[len(msgEnc)-1] = '\n'
	return msgEnc
}

func EncodeB64Message(cmd string, msg []byte) []byte {
	return EncodeMessage(cmd, msg, base64.StdEncoding)
}

func EncodePlainMessage(cmd string, msg []byte) []byte {
	return EncodeMessage(cmd, msg, echoEncoder{})
}

type Encoder interface {
	EncodedLen(int) int
	Encode([]byte, []byte)
}

type echoEncoder struct{}

func (b echoEncoder) EncodedLen(i int) int {
	return i
}

func (b echoEncoder) Encode(dst, src []byte) {
	copy(dst, src)
}

func ErrorHandler() chan<- error {
	errors := make(chan error)
	go func() {
		for err := range errors {
			log.Println(err)
		}
	}()
	return errors
}

type Connection struct {
	s    *Server
	send chan<- []byte
	recv <-chan byte
	queueName string
}

func (self *Server) Connection(send chan<- []byte, recv <-chan byte) *Connection {
	log.Println("new connection")
	return &Connection{
		s: self,
		send: send,
		recv: recv,
		queueName: "default",
	}
}

func (c *Connection) queue() Queue {
	return c.s.queues[c.queueName]
}

func (c *Connection) Serve() {
	defer c.Close()
	defer func() {
		if e := recover(); e != nil {
			c.send <- EncodeB64Message("error", []byte(fmt.Sprintf("%v", e)))
		}
	}()

	enque := c.Respond(c.Enque, base64.StdEncoding)
	deque := c.Respond(c.Deque, base64.StdEncoding)
	has := c.Respond(c.Has, echoEncoder{})
	size := c.Respond(c.Size, echoEncoder{})
	use := c.Respond(c.Use, echoEncoder{})
	badDecode := c.Respond(c.BadDecode, base64.StdEncoding)

	b64cmds := func(cmd string, data []byte) {
		switch cmd {
		case "ENQUE":
			enque(data)
		case "HAS":
			has(data)
		}
	}

	for line := range netutils.Readlines(c.recv) {
		command, rest := DecodeCmd(line)
		switch command {
		case "ENQUE", "HAS":
			if rest == nil {
				badDecode(rest)
			} else {
				data, err := DecodeB64(rest)
				if err != nil {
					badDecode(rest)
				} else {
					b64cmds(command, data)
				}
			}
		case "DEQUE":
			deque(rest)
		case "SIZE":
			size(rest)
		case "USE":
			use(rest)
		default:
			err := fmt.Errorf("bad command recieved, '%v'", command)
			log.Println(err.Error())
			c.send <- EncodeB64Message("ERROR", []byte(err.Error()))
		}
	}
}

func (c *Connection) Close() {
	close(c.send)
	<-c.recv
	log.Println("closed connection")
}

func (c *Connection) Respond(f func([]byte) (string, []byte, error), enc Encoder) (g func([]byte)) {
	return func(rest []byte) {
		cmd, data, err := f(rest)
		if err != nil {
			if err.Error() != "queue is empty" {
				log.Println(err)
			}
			c.send <- EncodeB64Message("ERROR", []byte(err.Error()))
		} else if cmd != "" {
			c.send <- EncodeMessage(cmd, data, enc)
		} else {
			c.send <- EncodePlainMessage("OK", nil)
		}
	}
}

func (c *Connection) BadDecode(line []byte) (string, []byte, error) {
	return "", nil, fmt.Errorf("bad line '%v'", string(bytes.TrimSpace(line)))
}

func (c *Connection) Use(rest []byte) (string, []byte, error) {
	if rest == nil {
		return "", nil, fmt.Errorf("Must supply a queue name")
	}
	name := strings.TrimSpace(string(rest))
	if name == "" {
		return "", nil, fmt.Errorf("Must supply a (non-blank) queue name")
	}
	if _, has := c.s.queues[name]; !has {
		c.s.queues[name] = c.s.newQueue()
	}
	c.queueName = name
	return "OK", nil, nil
}

func (c *Connection) Enque(rest []byte) (string, []byte, error) {
	if rest == nil {
		return "", nil, fmt.Errorf("no data sent to queue")
	} else {
		return "", nil, c.queue().Enque(rest)
	}
}

func (c *Connection) Has(rest []byte) (string, []byte, error) {
	if len(rest) != sha256.Size {
		return "", nil, fmt.Errorf("Expected a hash of size %v got %v", sha256.Size, len(rest))
	}
	if c.queue().Has(rest) {
		return "TRUE", nil, nil
	} else {
		return "FALSE", nil, nil
	}
}

func (c *Connection) Size(rest []byte) (string, []byte, error) {
	return "SIZE", []byte(fmt.Sprint(c.queue().Size())), nil
}

func (c *Connection) Deque(rest []byte) (string, []byte, error) {
	if rest != nil {
		return "", nil, fmt.Errorf("recieved msg data when none was expected")
	}
	if c.queue().Empty() {
		return "", nil, fmt.Errorf("queue is empty")
	}
	data, err := c.queue().Deque()
	if err != nil {
		return "", nil, err
	}
	return "ITEM", data, nil
}

