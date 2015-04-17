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

import "testing"

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
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

func TestEnqueThenDeque(t *testing.T) {
	q := NewQueue(true)
	l := make([][]byte, 0, 25)
	for i := 0; i < rand.Intn(25)+10; i++ {
		item := rand_bytes(rand.Intn(32) + 2)
		l = append(l, item)
		if err := q.Enque(item); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range l {
		if q.Empty() {
			t.Fatal("Queue should not be empty")
		}
		q_item, err := q.Deque()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(q_item, item) {
			t.Fatal("items should have equalled each other")
		}
	}
	q_item, err := q.Deque()
	if q_item != nil {
		t.Fatal("Item should have been nil")
	}
	if err == nil {
		t.Fatal("should have gotten an error")
	}
	if err.Error() != "List is empty" {
		t.Fatal("should have gotten empty list error")
	}
	if !q.Empty() {
		t.Fatal("queue should have been empty.")
	}
}
