# Queued - A simple queue daemon

By Tim Henderson (tim.tadh@gmail.com)

`queued` is a very simple network daemon which provides clients with a simple
line oriented ASCII protocol for interacting with a FIFO queue. There are
(beta) clients available for Python and Scala in the clients directory. Queued
only currently provides one queue and does not persist the queue or allow it to
grow larger than the amount the program can allocate on the machine (eg. there
is no disk cache).

## Docs

Install

    go get github.com/timtadh/queued

### Usage Docs

```
queued <port>

starts a queued daemon, a simple queue exposed on the network.

Options
    -h, --help                          print this message
    --allow-dups                        allow duplicate items in the queue

    Specs
        <port>
                A bindable port number.
```

### API Docs

On `godoc.org`:

[Documentation](http://godoc.org/github.com/timtadh/queued)

### Protocol


This package implements a TCP network service which allows clients to queue
and deque data. The client has the following verbs

- ENQUE
- DEQUE
- HAS
- SIZE
- USE

the server can send the following reponse status words

- OK
- ERROR
- ITEM
- TRUE
- FALSE
- SIZE

All messages have the following format:

    COMMAND DATA

COMMAND is a single ASCII word with no spaces. DATA is the rest of the line.
DATA is optional. For ERROR, ENQUE and ITEM it is Base64 encoded. For SIZE
reponse it is ASCII.

The client can send any command at any time. The server may at any command
respond with ERROR if there was a problem processing the command.

##### USE name

Use the named queue. You never have to issue this command. If you do not
you will automatically be using a queue named "default". If the queue does
not this command will create one. The server should respond:

     OK

name should have no spaces and should be utf8.

##### ENQUE XXXXXXXXXXXXXXXX

XXXXXXXXXXXXXXXXXX should be base64 encoded data. If it is not the
server will respond with and ERROR.  Otherwise the server will repond

    OK

The server should always respond with OK if the command is properly
formated. If it does not that means there is an internal error in the
server.

##### DEQUE

If the queue is empty the server will respond with:

    ERROR cXVldWUgaXMgZW1wdHk=

Which decodes too:

    ERROR queue is empty

Otherwise it repondes with

    ITEM XXXXXXXXXXXXXXX

##### HAS XXXXXXXXXXXXXXXXXXXXXXXXXXX

HAS checks for the existence of an item. It doesn't send the item but
rather the base64 encoded sha265sum. Example for producing this (in
Python for simplicity)

    >>> import hashlib
    >>> hashlib.sha256('item').digest().encode('base64').strip()
    'SjPqzV+mXysuKHHNExKGtTxBWxMWZtcRc7tuP+WTYbM='

The client would send

    HAS SjPqzV+mXysuKHHNExKGtTxBWxMWZtcRc7tuP+WTYbM=

The server would reponse either

    TRUE

or

    FALSE


##### SIZE

The server response with the size of the queue encoded as a base10
ascii number. eg.

    SIZE 9231

For a queue that is 9,231 items long.

