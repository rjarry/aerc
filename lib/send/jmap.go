package send

import (
	"fmt"
	"io"

	"github.com/emersion/go-message/mail"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

func newJmapSender(
	worker *types.Worker, from *mail.Address, rcpts []*mail.Address,
	copyTo []string,
) (io.WriteCloser, error) {
	var writer io.WriteCloser
	done := make(chan error)

	worker.PostAction(
		&types.StartSendingMessage{From: from, Rcpts: rcpts, CopyTo: copyTo},
		func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				return
			case *types.Unsupported:
				done <- fmt.Errorf("unsupported by worker")
			case *types.Error:
				done <- msg.Error
			case *types.MessageWriter:
				writer = msg.Writer
			default:
				done <- fmt.Errorf("unexpected worker message: %#v", msg)
			}
			close(done)
		},
	)

	err := <-done

	return writer, err
}
