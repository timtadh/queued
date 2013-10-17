/*
A simple daemon that exposes a queue as a network service on the port of your
choice. `queued` is made available under a BSD license.
*/

package main

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
  "os"
  "fmt"
  "strconv"
  "github.com/timtadh/getopt"
)

import (
  queue "github.com/timtadh/queued/queue"
  qnet "github.com/timtadh/queued/net"
)

var ErrorCodes map[string]int = map[string]int{
    "usage":1,
    "version":2,
    "opts":3,
    "badint":5,
}

var UsageMessage string = "queued <port>"
var ExtendedMessage string = `
starts a queued daemon, a simple queue exposed on the network.

Options
    -h, --help                          print this message

Specs
    <port>
        A bindable port number.
`

func Usage(code int) {
    fmt.Fprintln(os.Stderr, UsageMessage)
    if code == 0 {
        fmt.Fprintln(os.Stderr, ExtendedMessage)
        code = ErrorCodes["usage"]
    } else {
        fmt.Fprintln(os.Stderr, "Try -h or --help for help")
    }
    os.Exit(code)
}

func parse_int(str string) int {
    i, err := strconv.Atoi(str)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing '%v' expected an int\n", str)
        Usage(ErrorCodes["badint"])
    }
    return i
}

func main() {

    args, optargs, err := getopt.GetOpt(os.Args[1:], "h", []string{"help"})
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        Usage(ErrorCodes["opts"])
    }

    port := -1
    for _, oa := range optargs {
        switch oa.Opt() {
        case "-h", "--help": Usage(0)
        }
    }

    if len(args) != 1 {
        fmt.Fprintln(os.Stderr, "You must specify a port")
        Usage(ErrorCodes["opts"])
    }
    port = parse_int(args[0])

    fmt.Println("starting")
    server := qnet.NewServer(queue.NewQueue())
    server.Start(port)
}

