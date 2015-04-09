/*
This implements a simple linked list Queue. Nothing complicated. The queue is
thread safe and easy to use.
*/
package queue

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
	logpkg "log"
	"os"
	"strings"
	"sync"
)

var log *logpkg.Logger

func init() {
	log = logpkg.New(os.Stderr, "queued/queue> ", logpkg.Ltime|logpkg.Lshortfile)
}

type node struct {
	next *node
	data []byte
}

type Queue struct {
	head   *node
	tail   *node
	length int
	lock   *sync.Mutex
}

/* Construct a new queue */
func NewQueue() *Queue {
	return &Queue{
		head:   nil,
		tail:   nil,
		length: 0,
		lock:   new(sync.Mutex),
	}
}

/* Put data on the queue */
func (self *Queue) Enque(data []byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	node := &node{next: nil, data: data}

	if self.tail == nil {
		if self.head != nil {
			return fmt.Errorf("List has head but no tail...")
		}
		self.head = node
		self.tail = node
	} else {
		self.tail.next = node
		self.tail = node
	}
	self.length += 1

	return nil
}

/* Read data off the queue in FIFO order */
func (self *Queue) Deque() (data []byte, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.length == 0 {
		return nil, fmt.Errorf("List is empty")
	}
	if self.length < 0 {
		return nil, fmt.Errorf("List length is less than zero")
	}
	if self.head == nil {
		return nil, fmt.Errorf("head is nil")
	}

	node := self.head

	if node.next == nil {
		if node != self.tail {
			return nil, fmt.Errorf("Expected tail to equal head")
		}
		if self.length != 1 {
			return nil, fmt.Errorf("Expected list length to equal 1")
		}
		self.tail = nil
	}
	self.head = node.next
	self.length -= 1

	return node.data, nil
}

/* Check to see if it empty */
func (self *Queue) Empty() bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.length <= 0
}

/* How many items are on the queue? */
func (self *Queue) Length() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.length
}

func (self *node) String() string {
	return fmt.Sprintf("<node: '%s'>", string(self.data))
}

func (self *Queue) String() string {
	self.lock.Lock()
	defer self.lock.Unlock()

	var strs []string
	strs = append(strs, "<queue: ")
	for c := self.head; c != nil; c = c.next {
		strs = append(strs, c.String())
		if c.next != nil {
			strs = append(strs, ", ")
		}
	}
	strs = append(strs, ">")
	return strings.Join(strs, "")
}
