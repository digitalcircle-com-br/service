package service

import "context"

type Msg struct {
	Chan    string
	Payload string
	Err     error
}

func Sub(ch ...string) (ret <-chan *Msg, closefn func()) {
	sub := rediscli.Subscribe(context.Background(), ch...)
	inret := make(chan *Msg)
	ret = inret
	run := true
	go func() {
		for run {
			msg, err := sub.ReceiveMessage(context.Background())
			m := &Msg{Chan: msg.Channel, Payload: msg.Payload, Err: err}
			inret <- m
		}
	}()

	closefn = func() {
		run = false
		sub.Close()
		close(inret)
	}
	return
}

func Pub(ch string, msg interface{}) {
	rediscli.Publish(context.Background(), ch, msg)
}
