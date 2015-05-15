// This is a memory based buffer for simple key/value pairs where the
// pair will be removed after a specific time-to-live value. You can
// put a value in the buffer and read it before the TTL is reached.
// After this time the value is removed and every read operation will
// return nil.
package timebuffer

import "time"

var (
	buffer    map[string]interface{}
	requests  rqchan
	responses rspchan
)

type rq struct {
	key string
	val interface{}
}

type rsp struct {
	key string
	val chan interface{}
}
type rqchan chan rq
type rspchan chan rsp

func init() {
	buffer = make(map[string]interface{})
	requests = make(rqchan)
	responses = make(rspchan)

	go handle(requests, responses)
}

// Puts a value at 'key' with a time-to-live of ttl in seconds.
func Put(key string, val interface{}, ttl int) {
	r := rq{key, val}
	requests <- r

	if ttl > 0 {
		go func() {
			time.Sleep(time.Duration(ttl) * time.Second)
			r := rq{key, nil}
			requests <- r
		}()
	}
}

// Read the value at 'key'. It the TTL of the Put-operation is
// reached, this function returns nil.
func Get(key string) interface{} {
	r := rsp{key: key}
	r.val = make(chan interface{})

	responses <- r
	return <-r.val
}

func handle(rqc rqchan, rspc rspchan) {
	for {
		select {
		case rq := <-rqc:
			if rq.val == nil {
				delete(buffer, rq.key)
			} else {
				buffer[rq.key] = rq.val
			}
		case rsp := <-rspc:
			res, _ := buffer[rsp.key]
			rsp.val <- res
		}
	}
}
