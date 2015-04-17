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
	"crypto/sha256"
	"fmt"
	logpkg "log"
	"os"
	"strings"
	"sync"
)

import (
	"github.com/timtadh/data-structures/hashtable"
	"github.com/timtadh/data-structures/types"
)

var log *logpkg.Logger

func init() {
	log = logpkg.New(os.Stderr, "queued/queue> ", logpkg.Ltime|logpkg.Lshortfile)
}

// Generate a sha256 hash of the data
func Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return types.ByteSlice(h[:])
}

type node struct {
	next *node
	data []byte
}

type Queue struct {
	head   *node
	tail   *node
	length int
	index  *hashtable.LinearHash
	lock   *sync.Mutex
	allowDups bool
}

/* Construct a new queue */
func NewQueue(allowDups bool) *Queue {
	return &Queue{
		head:   nil,
		tail:   nil,
		length: 0,
		index: hashtable.NewLinearHash(),
		lock:   new(sync.Mutex),
		allowDups: allowDups,
	}
}

/* Put data on the queue */
func (self *Queue) Enque(data []byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	h := types.ByteSlice(Hash(data))
	has := self.index.Has(h)
	if !self.allowDups && has {
		return nil
	} else if has {
		i, err := self.index.Get(h)
		if err != nil {
			return err
		}
		err = self.index.Put(h, types.Int(int(i.(types.Int)) + 1))
		if err != nil {
			return err
		}
	} else {
		err := self.index.Put(h, types.Int(1))
		if err != nil {
			return err
		}
	}

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

	h := types.ByteSlice(Hash(node.data))
	if self.index.Has(h) {
		i, err := self.index.Get(h)
		if err != nil {
			return nil, err
		}
		j := int(i.(types.Int)) - 1
		if j <= 0 {
			_, err = self.index.Remove(h)
			if err != nil {
				return nil, err
			}
		} else {
			err = self.index.Put(h, types.Int(j))
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, fmt.Errorf("integrity error, index did not have data")
	}

	return node.data, nil
}

/* Check to see if it empty */
func (self *Queue) Empty() bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.length <= 0
}

/* How many items are on the queue? */
func (self *Queue) Size() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.length
}

func (self *Queue) Has(hash []byte) bool {
	if len(hash) != sha256.Size {
		return false
	}
	return self.index.Has(types.ByteSlice(hash))
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
