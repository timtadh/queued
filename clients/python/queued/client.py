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


import sys, os, socket, threading, time
from collections import deque

class Queue(object):

    def __init__(self, host, port, debug=False):
        self.host = host
        self.port = port
        self.debug = debug
        self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.conn.connect((self.host, self.port))
        self.queue_lock = threading.Lock()
        self.lines = deque()
        self.lines_cv = threading.Condition()
        self.read_thread = threading.Thread(target=self.listen)
        self.read_thread.daemon = True
        self.read_thread.start()

    def close(self):
        with self.queue_lock:
            self.conn.shutdown(socket.SHUT_RDWR)
            self.read_thread.join()
            if self.debug:
                print >>sys.stderr, "closed"

    def enque(self, data):
        with self.queue_lock:
            msg = "ENQUE " + data.encode('base64').replace('\n', '') + '\n'
            self.conn.send(msg)
            self.get_enque_response()

    def get_enque_response(self):
        cmd, data = self.get_line()
        if cmd == "ERROR":
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
            if data == "queue is empty":
                raise IndexError(data)
            raise Exception(data)
        elif cmd != "ITEM":
            raise Exception, "bad command recieved %s" % cmd
        assert cmd == "ITEM"
        return data

    def listen(self):
        chunk = ''
        while True:
            while "\n" not in chunk:
                try:
                    data = self.conn.recv(4096*4)
                    if not data: return
                    chunk += data
                except Exception, e:
                    print >>sys.stderr, e
                    return
            line, chunk = chunk.split('\n', 1)
            with self.lines_cv:
                self.lines.append(line)
                self.lines_cv.notify()

    def get_line(self):
        with self.lines_cv:
            while len(self.lines) <= 0:
                self.lines_cv.wait()
            line = self.lines.popleft()
        return self.process_line(line)

    def process_line(self, line):
        split = line.split(' ', 1)
        command = split[0]
        rest = None
        if len(split) > 1:
            rest = split[1].decode('base64')

        return command, rest

def _loop(queue):
    while True:
        try:
            line = raw_input('> ')
        except:
            break
        split = line.split(' ', 1)
        command = split[0]
        data = None
        if len(split) > 1:
            data = split[1]
        if command == 'enque' and data is not None:
            queue.enque(data)
        elif command == 'enque' and data is None:
            print >>sys.stderr, "bad input, need data"
        elif command == "deque":
            try:
                print queue.deque()
            except IndexError:
                print "queue is empty"

def main():
    queue = Queue('localhost', 9001)
    try:
        _loop(queue)
    finally:
        queue.close()

if __name__ == '__main__':
    main()

