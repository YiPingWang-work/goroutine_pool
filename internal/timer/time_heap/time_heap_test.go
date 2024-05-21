package time_heap

import (
	"context"
	"fmt"
	"goroutine_pool/log_plus"
	"log"
	"testing"
	"time"
)

func TestTimeHeap(t *testing.T) {
	log_plus.Grade = log_plus.DEBUG_TIME_HEAP
	th := &TimeHeap{}
	th.Init(false)
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		for {
			select {
			case <-th.C():
				if err := th.Callback(); err != nil {
					panic(err)
				}
			case <-ctx.Done():
				fmt.Println("over")
				return
			}
		}
	}(ctx)
	log.Println()
	for i := 0; i < 10; i++ {
		i := i
		time.Sleep(time.Second)
		th.Register(2*time.Second, func([]any) error {
			log.Println("hello", i)
			return nil
		}, nil, -1)
	}
	time.Sleep(3 * time.Second)
	cancel()
}
