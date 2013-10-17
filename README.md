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

### Command Docs

```
queued <port>

starts a queued daemon, a simple queue exposed on the network.

Options
    -h, --help                          print this message

    Specs
        <port>
                A bindable port number.
```

### API Docs

On `godoc.org`:

[Documentation](http://godoc.org/github.com/timtadh/queued)

