package time_heap

import (
	"goroutine_pool/log_plus"
	"goroutine_pool/monitor"
	"sync"
	"time"
)

import (
	"container/heap"
)

var _ heap.Interface = (*eventHeap)(nil)

type TimeHeap struct {
	*time.Timer
	heap         eventHeap
	activeEvents map[int64]*event
	autoKey      int64
	m            sync.Mutex
	needm        bool
}

type event struct {
	key    int64
	when   time.Time
	fn     func([]any) error
	params []any
}

type eventHeap []*event

func (t *TimeHeap) Init(async bool) {
	never := &event{
		key:  0,
		when: time.Now().Add(1e5 * time.Hour),
		fn: func([]any) error {
			panic("timeHeap need reset")
		},
		params: nil,
	}
	t.needm = async
	t.activeEvents = map[int64]*event{0: never}
	heap.Push(&t.heap, never)
	t.Timer = time.NewTimer(1e5 * time.Hour)
}

func (t *TimeHeap) Register(after time.Duration, f func([]any) error, params []any, key int64) int64 {
	after = max(after, time.Nanosecond)
	t.lock()
	defer t.unlock()
	when := time.Now().Add(after)
	if key == 0 {
		t.autoKey++
		key = t.autoKey
	} else if key < 0 {
		if olde, has := t.activeEvents[key]; has && when.UnixNano() >= olde.when.UnixNano() {
			olde.fn = func([]any) error {
				t.unlock()
				defer t.lock()
				t.Register(when.Sub(time.Now()), f, params, key)
				log_plus.Printf(log_plus.DEBUG_TIME_HEAP, "timeHeap: event %d changed, process failed\n", key)
				return nil
			}
			return key
		}
	} else if _, has := t.activeEvents[key]; has {
		panic("timeHeap: repeated event key")
	}
	e := &event{
		key:    key,
		when:   when,
		fn:     f,
		params: params,
	}
	t.activeEvents[key] = e
	if when.UnixNano() < t.heap.Top().(*event).when.UnixNano() {
		t.Reset(after)
	}
	heap.Push(&t.heap, e)
	log_plus.Printf(log_plus.DEBUG_TIME_HEAP, "timeHeap: register event: %d\n", e.key)
	return e.key
}

func (t *TimeHeap) Remove(key int64) {
	t.lock()
	defer t.unlock()
	delete(t.activeEvents, key)
	log_plus.Printf(log_plus.DEBUG_TIME_HEAP, "timeHeap: remove event: %d\n", key)
}

func (t *TimeHeap) Callback() (err error) {
	for {
		monitor.AddLoops()
		t.lock()
		if time.Now().UnixNano() < t.heap.Top().(*event).when.UnixNano() {
			t.Reset(t.heap.Top().(*event).when.Sub(time.Now()))
			break
		}
		top := heap.Pop(&t.heap).(*event)
		if _, ok := t.activeEvents[top.key]; ok {
			delete(t.activeEvents, top.key)
			log_plus.Printf(log_plus.DEBUG_TIME_HEAP, "timeHeap: process event: %d\n", top.key)
			if err = top.fn(top.params); err != nil {
				break
			}
		}
		t.unlock()
	}
	t.unlock()
	return
}

func (t *TimeHeap) C() <-chan time.Time {
	return t.Timer.C
}

func (t *TimeHeap) lock() {
	if t.needm {
		t.m.Lock()
	}
}

func (t *TimeHeap) unlock() {
	if t.needm {
		t.m.Unlock()
	}
}

// 继承heap接口

func (t eventHeap) Len() int {
	return len(t)
}

func (t eventHeap) Less(i, j int) bool {
	return t[i].when.UnixNano() < t[j].when.UnixNano()
}

func (t eventHeap) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t *eventHeap) Push(x any) {
	*t = append(*t, x.(*event))
}

func (t *eventHeap) Pop() any {
	x := (*t)[len(*t)-1]
	*t = (*t)[:len(*t)-1]
	return x
}

func (t eventHeap) Top() any {
	return t[0]
}
