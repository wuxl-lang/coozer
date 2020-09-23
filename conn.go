package coozer

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"google.golang.org/protobuf/proto"
)

// txn indicates one transaction which holds a request and response in one interaction
type txn struct {
	id   string //An identifier to distingusih different transaction
	req  Request
	resp *Response
	err  error     // it indicates an error happnes
	done chan bool // it indicates the interactin is done with either an error or a response
}

// Conn represents a connect to a server
type Conn struct {
	addr    string      // address of the server
	conn    net.Conn    // connection
	send    chan *txn   // pointer of a transaction with request are placed to the channel and waits for sending to server
	msg     chan []byte // message which recieves from server
	err     error       // it indicates an error happens in the connection, it should be released
	stop    chan bool   // its the signal to stop the connection
	stopped chan bool
}

// Dail method tries to connect a server and return a Conn
func Dail(addr string) (*Conn, error) {
	return dail(addr, -1)
}

// DailTimeout tries to connect a server with a timeout
func DailTimeout(addr string, timeout int32) (*Conn, error) {
	return dail(addr, -1)
}

// Close indicates close the connection
func (c *Conn) Close() error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	return nil
}

func dail(addr string, timeout time.Duration) (*Conn, error) {
	var c Conn
	var err error

	c.addr = addr
	// connect TCP connection
	if timeout > 0 {
		c.conn, err = net.DialTimeout("tcp", addr, timeout)
	} else {
		c.conn, err = net.Dial("tcp", addr)
	}

	if err != nil {
		return nil, err
	}

	c.send = make(chan *txn)
	c.msg = make(chan []byte)
	c.stop = make(chan bool)
	c.stopped = make(chan bool)

	// An external error channel to stop the connection when execption happens
	errch := make(chan error, 1)

	// A gorountine to handle request from client
	go c.mux(errch)
	// A gorountine to keep on reading message from server
	go c.readAll(errch)

	return &c, nil
}

// handle request from client
func (c *Conn) mux(errch chan error) {
	// Local state, a map contains a id to a txt pointer
	txns := make(map[int32]*txn)
	var n int32
	var err error

	// main loop
	for {
		select {
		case txn := <-c.send: // when it receives reqesut from client
			fmt.Println("Recieve txn in send channel with id: " + txn.id)
			// Put new transaction into the local state
			// Very simple strategy to assign an index to it
			for txns[n] != nil {
				n++
			}

			txns[n] = txn
			txn.req.Tag = n

			// Marshal request in transaction
			var buf []byte
			if buf, err = proto.Marshal(&txn.req); err != nil {
				fmt.Println("Unable to marshal request ignore txn with id: ", txn.id)

				txns[n] = nil    // clean state
				txn.err = err    // update err in transaction
				txn.done <- true // transaction is finished
				continue
			}

			// write requset to server
			if err = c.write(buf); err != nil {
				fmt.Println("Failed to write bytes via TCP for txn with id: ", txn.id, ", connection will be released")
				goto failure
			}

			fmt.Println("Finished to write buf for txn with id: ", txn.id)
		case buf := <-c.msg: // when server return response
			var resp Response
			fmt.Println("Receive message ", string(buf))

			if err = proto.Unmarshal(buf, &resp); err != nil {
				fmt.Println("Failed to unmarshal message to response", err)
				continue
			}

			txn, ok := txns[resp.Tag]
			if !ok {
				fmt.Println("Can't find txn according to tag in resp, ignore the resp ", resp.Tag)
				continue
			}

			// Clean a completed transaction
			delete(txns, resp.Tag)
			txn.resp = &resp
			txn.done <- true // transaction is finished
		case err = <-errch: // when an error happens
			goto failure
		case <-c.stop: // when it is required to close
			err = ErrClosed
			goto failure
		}
	}

	// It is failed to handle write/read from the connection.
	// All transactions are marked as done and close connection.
failure:
	c.err = err
	for _, txn := range txns {
		// clean all the in-process txn
		txn.err = err
		txn.done <- true
	}

	// close the connection
	c.conn.Close()
	close(c.stopped)
}

func (c *Conn) write(buf []byte) error {
	if err := binary.Write(c.conn, binary.BigEndian, int32(len(buf))); err != nil {
		return err
	}

	_, err := c.conn.Write(buf)
	return err
}

func (c *Conn) readAll(errch chan error) {
	for {
		fmt.Println("Retry to readAll")
		buf, err := c.read()
		if err != nil {
			errch <- err // failed to read message from server, send error message to close the connection
			return
		}

		c.msg <- buf
	}
}

func (c *Conn) read() ([]byte, error) {
	var size int32
	err := binary.Read(c.conn, binary.BigEndian, &size)
	if err != nil {
		return nil, err
	}

	fmt.Println("Read from server with size ", size)

	buf := make([]byte, size)
	_, err = io.ReadFull(c.conn, buf)
	if err != nil {
		return nil, err
	}

	fmt.Println("Read content from server ", string(buf))

	return buf, nil
}

func (c *Conn) call(t *txn) error {
	t.done = make(chan bool)
	select {
	case <-c.stopped: // When connection is closed 
		return c.err
	case c.send <- t:
		fmt.Println("Send txn to send channel in connection")

		// wait until the connection is stoped or transaction is done
		select {
		case <-c.stopped:
			return c.err
		case <-t.done:
			fmt.Println("It is done")
			if t.err != nil {
				return t.err
			}

			if t.resp.ErrCode != Response_NIL {
				return nil //
			}
		}
	}

	return nil
}

// Set body to content of a file, if it hasn't been modified since old revision
func (c *Conn) Set(file string, oldRev int64, body []byte) (newRev int64, err error) {
	var t txn
	t.req.Verb = Request_SET
	t.req.Path = file
	t.req.Rev = oldRev
	t.req.Value = body

	if err := c.call(&t); err != nil {
		return 0, err
	}

	return 0, nil
}

// Del deletes file, if it hasn't been modified
func (c *Conn) Del(file string, rev int64) error {
	var t txn
	t.req.Verb = Request_DEL
	t.req.Path = file
	t.req.Rev = rev

	if err := c.call(&t); err != nil {
		return err
	}

	return nil
}

// Get returns body, revision of the file at path
func (c *Conn) Get(file string, rev int64) ([]byte, int64, error) {
	fmt.Println("Get file ", file, " with revision ", rev)
	var t txn

	t.req.Verb = Request_GET
	t.req.Path = file
	t.req.Rev = rev 

	if err := c.call(&t); err != nil {
		return nil, 0, err
	}

	return t.resp.Value, t.resp.Rev, nil
}

// Self returns the node's identifier
func (c *Conn) Self() ([]byte, error) {
	var t txn
	t.req.Verb = Request_SELF

	if err := c.call(&t); err != nil {
		return nil, err
	}

	return t.resp.Value, nil
}

// Rev returns the current revision of the store
func (c *Conn) Rev() (int64, error) {
	var t txn
	t.req.Verb = Request_REV

	if err := c.call(&t); err != nil {
		return 0, err
	}

	return t.resp.Rev, nil
}

// Wait for the first change, on or after rev, to any file matching glob.
func (c *Conn) Wait(glob string, rev int64) (event Event, err error) {
	var t txn
	t.req.Verb = Request_WAIT
	t.req.Path = glob
	t.req.Rev = rev

	// https://yourbasic.org/golang/named-return-values-parameters/
	if err = c.call(&t); err != nil {
		return
	}

	event.Rev = t.resp.Rev
	event.Path = t.resp.Path
	event.Body = t.resp.Value
	event.Flag = t.resp.Flags & (set | del)

	return
}
