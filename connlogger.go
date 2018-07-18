package grpc

import (
	"context"
	"log"

	"google.golang.org/grpc"

	"google.golang.org/grpc/connectivity"

	"errors"
	"sync"
)

var (
	ErrExists = errors.New("connect with name already exists")
)

// ConnLogger will hold grpc connection states and log on their state changes
type ConnLogger struct {
	// lock for protecting connMap
	connM sync.RWMutex
	// a map of connections currently being logged
	connMap map[string]*grpc.ClientConn
	// lock for protecting stateMap
	stateM sync.RWMutex
	// a map associating connections with their last state
	stateMap map[string]*connectivity.State
}

func NewConnLogger() *ConnLogger {
	connMap := make(map[string]*grpc.ClientConn, 0)
	stateMap := make(map[string]*connectivity.State)

	return &ConnLogger{
		connM:    sync.RWMutex{},
		connMap:  connMap,
		stateM:   sync.RWMutex{},
		stateMap: stateMap,
	}
}

func (c *ConnLogger) RemoveConn(name string) {
	// take write lock on conn map and remove
	c.connM.Lock()
	delete(c.connMap, name)
	c.connM.Unlock()
}

// AddConn adds a connection to our ConnLogger
func (c *ConnLogger) AddConn(name string, conn *grpc.ClientConn) error {
	// take readlock to see if name exists
	c.connM.RLock()
	_, ok := c.connMap[name]
	c.connM.RUnlock()
	if ok {
		return ErrExists
	}

	// take write lock on connMap.
	c.connM.Lock()
	// confirm again that the name is not taken
	_, ok = c.connMap[name]
	if ok {
		c.connM.Unlock()
		return ErrExists
	}
	// add to map and return
	c.connMap[name] = conn
	c.connM.Unlock()

	// add current state to state map
	c.stateM.Lock()
	cs := conn.GetState()
	c.stateMap[name] = &cs
	c.stateM.Unlock()

	// start logConn go routine
	go c.logConn(name)

	// log
	log.Printf("ConnLogger: connection for %s is being logged. current state %s", name, cs)

	return nil
}

// logConn is ran as a go routine when a connection is added. it begins tracking
// the connection state and logging state changes to stdout
func (c *ConnLogger) logConn(name string) {
	// create useless context.Background() here to not allocate on every iteration.
	// TODO: plumb a real context into here that when cancels kills this for loop
	ctx := context.Background()
	for {
		// get read lock to determine of connection exists. if connection does not exist return
		c.connM.RLock()
		conn, ok := c.connMap[name]
		c.connM.RUnlock()
		if !ok {
			return
		}

		// read lock on state map to get last state
		c.connM.RLock()
		// we can be sure this won't be a nil pointer due to only calling this
		// method from the AddConn method
		state := c.stateMap[name]
		c.connM.RUnlock()

		// block on state change see: https://godoc.org/google.golang.org/grpc#ClientConn.WaitForStateChange
		res := conn.WaitForStateChange(ctx, *state)

		// immediately store state which triggered change
		newState := conn.GetState()

		// if res is true WaitForStateChange stopped blocking due to a connection state change. log
		if res {
			log.Printf("ConnLogger: state change for %s has occured. new state: %s", name, newState)
		}

		// update state map with recorded state
		c.stateM.Lock()
		c.stateMap[name] = &newState
		c.stateM.Unlock()

	}
}
