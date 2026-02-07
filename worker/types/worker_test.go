package types

import (
	"context"
	"testing"
	"time"
)

func TestWorkerCallback(t *testing.T) {
	worker := NewWorker("test", make(chan WorkerMessage))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			select {
			case action := <-worker.Actions():
				response := Message{
					inResponseTo: action,
					id:           2,
				}
				worker.ProcessMessage(&response)
			case <-ctx.Done():
				return

			}
		}
	}()

	msg := Message{id: 1}

	called := make(chan struct{})
	worker.PostAction(context.TODO(), &msg, func(msg WorkerMessage) {
		close(called)
	})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Errorf("callback was not called")
	}
}
