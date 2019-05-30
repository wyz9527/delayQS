package delayQS

import (
	"os"
	"os/signal"
	"syscall"
)

func signalStop(c chan<- os.Signal) {
	signal.Stop(c)
}

func signals() [2]chan bool{
	var quits [2]chan bool
	quits[0] = make( chan bool )
	quits[1] = make( chan bool )



	go func() {
		signals := make(chan os.Signal)
		defer close(signals)

		signal.Notify(signals, syscall.SIGQUIT, syscall.SIGTERM, os.Interrupt)
		defer signalStop(signals)

		<-signals
		quits[0] <- true
		quits[1] <- true
	}()

	return quits
}
