package main

import (
	raw "github.com/buger/gor/raw_socket_listener"
	"log"
	"net"
	"time"
)

// RAWInput used for intercepting traffic for given address
type RAWInput struct {
	data     chan *raw.TCPMessage
	address  string
	expire   time.Duration
	quit     chan bool
	engine   int
	listener *raw.Listener
}

// Available engines for intercepting traffic
const (
	EngineRawSocket = 1 << iota
	EnginePcap
)

// NewRAWInput constructor for RAWInput. Accepts address with port as argument.
func NewRAWInput(address string, engine int, expire time.Duration) (i *RAWInput) {
	i = new(RAWInput)
	i.data = make(chan *raw.TCPMessage)
	i.address = address
	i.expire = expire
	i.engine = engine
	i.quit = make(chan bool)

	i.listen(address)
	i.listener.IsReady()

	return
}

func (i *RAWInput) Read(data []byte) (int, error) {
	msg := <-i.data
	buf := msg.Bytes()

	var header []byte

	if msg.IsIncoming {
		header = payloadHeader(RequestPayload, msg.UUID(), msg.Start.UnixNano())
	} else {
		header = payloadHeader(ResponsePayload, msg.UUID(), msg.End.UnixNano()-msg.AssocMessage.Start.UnixNano())
	}

	copy(data[0:len(header)], header)
	copy(data[len(header):], buf)

	return len(buf) + len(header), nil
}

func (i *RAWInput) listen(address string) {
	Debug("Listening for traffic on: " + address)

	host, port, err := net.SplitHostPort(address)

	if err != nil {
		log.Fatal("input-raw: error while parsing address", err)
	}

	i.listener = raw.NewListener(host, port, i.engine, i.expire)

	ch := i.listener.Receiver()

	go func(){
		for {
			select {
			case <-i.quit:
				return
			default:
			}

			// Receiving TCPMessage object
			m := <-ch

			i.data <- m
		}
	}()
}

func (i *RAWInput) String() string {
	return "Intercepting traffic from: " + i.address
}

func (i *RAWInput) Close() {
	i.listener.Close()
	close(i.quit)
}
