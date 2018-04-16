package metadata

import (
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	// ErrCollectorClosed ...
	ErrCollectorClosed = errors.New("collector colsed")
	// ErrGetPiecesTimeout ...
	ErrGetPiecesTimeout = errors.New("get pieces timeout")
	// ErrRejectByPeer ...
	ErrRejectByPeer = errors.New("reject by peer")
	// ErrTooMuchPieces ...
	ErrTooMuchPieces = errors.New("too much peices")
)

// Collector collects metadata.
type Collector struct {
	onSuccess Then
	onError   Reject

	closeQ  chan struct{}
	queries chan *metadataQuery
	reply   chan struct{}

	MaxPendingQueries int
}

// NewCollector creates and return a new metadata collector.
func NewCollector() *Collector {
	var (
		maxPendingQueries = 10000
		queriesQsize      = 5000
	)
	c := Collector{
		MaxPendingQueries: maxPendingQueries,

		closeQ:  make(chan struct{}),
		queries: make(chan *metadataQuery, queriesQsize),
		reply:   make(chan struct{}, maxPendingQueries),
	}

	c.onSuccess = func(req *Request, meta *Metadata) {
		magnetLink := fmt.Sprintf("magnet:?xt=urn:btih:%s", req.HashInfo)
		log.Println("[collect onMetadata]", magnetLink, meta.Name)
	}

	c.onError = func(req *Request, err error) {
		log.Println("[collect onError]", err.Error())
	}

	go c.loop()

	return &c
}

func (c *Collector) loop() {
	var pendingQueries = 0

	for {
		if pendingQueries >= c.MaxPendingQueries {
			select {
			case <-c.closeQ:
				return
			case <-c.reply:
				pendingQueries--
			}
		} else {
			select {
			case <-c.closeQ:
				return
			case query := <-c.queries:
				pendingQueries++
				go query.start(c)
			case <-c.reply:
				pendingQueries--
			}
		}
	}
}

// Close the collector.
func (c *Collector) Close() {
	close(c.closeQ)
	close(c.queries)
	close(c.reply)
}

// GetSync adds a new query to the collector's queries queue and wait
// until then query finished.
func (c *Collector) GetSync(req *Request, then Then, reject Reject) error {
	if then == nil {
		then = c.onSuccess
	}
	if reject == nil {
		reject = c.onError
	}

	query := newMetadataQuery(req, then, reject)
	putTimeout := time.After(30 * time.Second)
	select {
	case <-c.closeQ:
		return ErrCollectorClosed
	case <-putTimeout:
		return errors.New("wait timeout")
	case <-c.queries:
	}

	query.wait()
	return nil
}

// Get adds a new query to the collector's queries queue.
// returns a error when queue is full.
func (c *Collector) Get(req *Request) error {
	query := newMetadataQuery(req, c.onSuccess, c.onError)
	select {
	case c.queries <- query:
	default:
		return errors.New("queries queue is full")
	}

	return nil
}

// OnFinish registers metadata handler.
func (c *Collector) OnFinish(handler Then) {
	log.Println("set collector handler")
	c.onSuccess = handler
}

// OnError registers error handler.
func (c *Collector) OnError(errHandler Reject) {
	c.onError = errHandler
}
