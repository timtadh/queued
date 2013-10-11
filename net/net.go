/*
This package implements a TCP network service which allows clients to queue and
deque data. The client has 2 verbs "ENQUE" and "DEQUE" and the server can send 3
reponse status words "OK", "ERROR" and "ITEM". 

All messages have the following format:

    COMMAND DATA

Where DATA is Base64 encoded and COMMAND is a single ASCII word with no spaces.

The client can send ENQUE or DEQUE at any time. If the queue is empty the server
will respond with:

    ERROR cXVldWUgaXMgZW1wdHk=

Which decodes too:

    ERROR queue is empty

Otherwise it repondes with

    ITEM XXXXXXXXXXXXXXX

When enqueing data the client sends

    ENQUE XXXXXXXXXXXXXXXXX

The server will respond with

    OK

Unless there is some internal error.

*/

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
  "fmt"
  "os"
  "net"
  "encoding/base64"
  "bytes"
  logpkg "log"
)

import (
  netutils "github.com/timtadh/netutils"
)


var log *logpkg.Logger

func init() {
    log = logpkg.New(os.Stderr, "queued/net> ", logpkg.Ltime | logpkg.Lshortfile)
}

type Server struct {
    ln *net.TCPListener
    queue Queue
}


func NewServer(queue Queue) *Server {
    return &Server{queue:queue}
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
      &net.TCPAddr{IP:net.ParseIP("0.0.0.0"), Port:port},
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
    var EOF bool
    for !EOF {
        con, err := self.ln.AcceptTCP()
        if netutils.IsEOF(err) {
            EOF = true
        } else if err != nil {
            log.Panic(err)
        } else {
            send := netutils.TCPWriter(con)
            recv := netutils.TCPReader(con)
            go self.connection(send, recv)
        }
    }
}


/*
Decode messages to/from the server. The server reads and sends line oriented
commands with message bodies base64 encoded. This function handles decoding
those lines for you.  */
func DecodeMessage(line []byte) (string, []byte) {
    split := bytes.SplitN(line, []byte(" "), 2)
    command := string(bytes.TrimSpace(split[0]))
    var rest []byte
    if len(split) > 1 {
        b64 := base64.StdEncoding
        encoded := bytes.TrimSpace(split[1])
        rest = make([]byte, b64.DecodedLen(len(encoded)))
        if n, err := b64.Decode(rest, encoded); err != nil {
            panic(err)
        } else {
            rest = rest[:n]
        }
    }
    return command, rest
}


/*
Encode a message for the server. Handles base64ing the msg. It is expected that
the paramter `cmd` will not have a space in it. If it does problems will occur
on the reciever side.  */
func EncodeMessage(cmd string, msg []byte) []byte {
    if msg == nil {
        return []byte(cmd + "\n")
    }
    b64 := base64.StdEncoding
    bcmd := []byte(cmd)
    cmdlen := len(bcmd)
    msg64len := cmdlen + b64.EncodedLen(len(msg)) + 2
    msg64 := make([]byte, msg64len)
    copy(msg64[:cmdlen], bcmd)
    msg64[cmdlen] = ' '
    b64.Encode(msg64[cmdlen+1:msg64len-1], msg)
    msg64[len(msg64)-1] = '\n'
    return msg64
}


func (self *Server) connection(send chan<- []byte, recv <-chan byte) {
    defer func() { <-recv }()
    defer log.Println("client closed")
    defer close(send)

    log.Println("new client")

    responder := func(f func([]byte)(string, []byte, error)) (g func([]byte)) {
        return func(rest []byte) {
            cmd, data, err := f(rest)
            if err != nil {
                log.Println(err)
                send<-EncodeMessage("ERROR", []byte(err.Error()))
            } else if cmd != "" {
                send<-EncodeMessage(cmd, data)
            } else {
                send<-EncodeMessage("OK", nil)
            }
        }
    }

    bad := responder(func(rest []byte) (string, []byte, error) {
        return "", nil, fmt.Errorf("bad command recieved")
    })

    enque := responder(func(rest []byte) (string, []byte, error) {
        // log.Println("got", rest)
        if rest == nil {
            return "", nil, fmt.Errorf("no data sent to queue")
        } else {
            return "", nil, self.queue.Enque(rest)
        }
    })

    deque := responder(func(rest []byte) (string, []byte, error) {
        if rest != nil {
            return "", nil, fmt.Errorf("recieved msg data when none was expected")
        }
        if self.queue.Empty() {
            return "", nil, fmt.Errorf("queue is empty")
        }
        data, err := self.queue.Deque()
        if err != nil {
            return "", nil, err
        }
        return "ITEM", data, nil
    })

    defer func() {
        if e := recover(); e != nil {
            send<-EncodeMessage("error", []byte(fmt.Sprintf("%v", e)))
        }
    }()

    for line := range netutils.Readlines(recv) {
        command, rest := DecodeMessage(line)
        switch command {
        case "ENQUE": enque(rest)
        case "DEQUE": deque(rest)
        default:
            log.Printf("'%s'\n", command)
            bad(rest)
        }
    }
}


