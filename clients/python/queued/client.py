#!/usr/bin/env python

'''
queued python client
Author: Tim Henderson
Email: tadh@case.edu
Copyright 2013 All Right Reserved

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

 * Redistributions of source code must retain the above copyright notice,
   this list of conditions and the following disclaimer.

 * Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

 * Neither the name of the queued nor the names of its contributors may be
   used to endorse or promote products derived from this software without
   specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
'''

import sys
import os
import socket
import threading
import time
import hashlib
import binascii
from collections import deque


class Queue(object):

    def __init__(self, host, port, debug=False):
        self.host = host
        self.port = port
        self.debug = debug
        self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.conn.settimeout(5)
        self.conn.connect((self.host, self.port))
        self.queue_lock = threading.Lock()
        self.lines = deque()
        self.lines_cv = threading.Condition()
        self.closed = False
        self.read_thread = threading.Thread(target=self.listen)
        self.read_thread.daemon = True
        self.read_thread.start()

    def close(self):
        self._close(False)

    def _close(self, from_read=False):
        if self.debug and from_read:
            print >>sys.stderr, "read thread closing it"
        with self.queue_lock:
            with self.lines_cv:
                if self.closed:
                    if self.debug and from_read:
                        print >>sys.stderr, "read thread bailed"
                    return
                self.closed = True
                self.lines_cv.notifyAll()
        self.conn.shutdown(socket.SHUT_RDWR)
        if not from_read:
            self.read_thread.join()
        self.conn.close()
        if self.debug:
            print >>sys.stderr, "closed"
            if from_read:
                print >>sys.stderr, "read thread closed it"

    def size(self):
        with self.queue_lock:
            msg = "SIZE\n"
            self.conn.send(msg)
            return self.get_size_response()

    def get_size_response(self):
        cmd, data = self.get_line()
        if cmd == "ERROR":
            data = data.decode('base64')
            raise Exception(data)
        elif cmd != "SIZE":
            raise Exception, "bad command recieved %s" % cmd
        else:
            return int(data)

    def hash(self, data):
        return hashlib.sha256(data).digest().encode('base64').strip()

    def has_hash(self, hash):
        try:
            hash.decode('base64')
        except binascii.Error, e:
            raise Exception("Hash must be base64 encoded. Use the hash function")
        h = hash.strip()
        with self.queue_lock:
            msg = "HAS " + h + "\n"
            self.conn.send(msg)
            return self.get_has_response()

    def has(self, data):
        h = self.hash(data)
        with self.queue_lock:
            msg = "HAS " + h + "\n"
            self.conn.send(msg)
            return self.get_has_response()

    def get_has_response(self):
        cmd, data = self.get_line()
        if cmd == "ERROR":
            data = data.decode('base64')
            raise Exception(data)
        elif cmd == "TRUE":
            return True
        elif cmd == "FALSE":
            return False
        else:
            raise Exception("bad server response %s %s" % (cmd, data))

    def enque(self, data):
        with self.queue_lock:
            msg = "ENQUE " + data.encode('base64').replace('\n', '') + '\n'
            self.conn.send(msg)
            self.get_enque_response()

    def get_enque_response(self):
        cmd, data = self.get_line()
        if cmd == "ERROR":
            data = data.decode('base64')
            raise Exception(data)
        elif cmd != "OK":
            raise Exception, "bad command recieved %s" % cmd

    def deque(self):
        with self.queue_lock:
            self.conn.send("DEQUE\n")
            return self.get_deque_response()

    def get_deque_response(self):
        cmd, data = self.get_line()
        if cmd == "ERROR":
            data = data.decode('base64')
            if data == "queue is empty":
                raise IndexError(data)
            raise Exception(data)
        elif cmd != "ITEM":
            raise Exception, "bad command recieved %s" % cmd
        assert cmd == "ITEM"
        data = data.decode('base64')
        return data

    def listen(self):
        chunk = ''
        while True:
            while "\n" not in chunk:
                try:
                    data = self.conn.recv(4096*4)
                    if not data:
                        self._close(True)
                        return
                    chunk += data
                except socket.timeout, t:
                    ## timeout retry
                    pass
                except Exception, e:
                    print >>sys.stderr, e, type(e)
                    self._close(True)
                    return
            line, chunk = chunk.split('\n', 1)
            with self.lines_cv:
                self.lines.append(line)
                self.lines_cv.notify()

    def get_line(self):
        with self.lines_cv:
            while len(self.lines) <= 0:
                if self.closed:
                    raise Exception, "queued connection closed"
                self.lines_cv.wait()
            line = self.lines.popleft()
        return self.process_line(line)

    def process_line(self, line):
        split = line.split(' ', 1)
        command = split[0]
        rest = None
        if len(split) > 1:
            rest = split[1].strip()

        return command, rest

def _loop(queue):
    while True:
        try:
            line = raw_input('> ')
        except:
            break
        split = line.split(' ', 1)
        command = split[0].lower().strip()
        data = None
        if len(split) > 1:
            data = split[1]
        if command == 'enque' and data is not None:
            queue.enque(data)
        elif command == 'enque' and data is None:
            print >>sys.stderr, "bad input, need data"
        elif command == "has" and data is not None:
            print queue.has(data)
        elif command == 'has' and data is None:
            print >>sys.stderr, "bad input, need data"
        elif command == "size":
            print queue.size()
        elif command == "deque":
            try:
                print queue.deque()
            except IndexError:
                print "queue is empty"
        elif command == '':
            pass
        else:
            print >>sys.stderr, "bad command"

def main():
    queue = Queue('localhost', 9001)
    try:
        _loop(queue)
    finally:
        queue.close()

if __name__ == '__main__':
    main()

