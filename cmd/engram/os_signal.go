package main

import (
	"os"
	"os/signal"

	"github.com/toejough/engram/internal/cli"
)

// unexported constants.
const (
	signalChannelBuffer = 10 // buffer so a burst of signals is not dropped
)

// forwardAsPulses converts OS signal deliveries into unit pulses.
func forwardAsPulses(sigCh <-chan os.Signal, pulses chan<- struct{}) {
	for range sigCh {
		pulses <- struct{}{}
	}
}

// registerForceExit subscribes to sigs and starts the force-exit watcher:
// the first signal is left to targ's context cancellation for graceful
// shutdown; the second forces exitFn (cli.ForceExitOnRepeatedSignal). Must
// run BEFORE targ.Main so the handler covers the whole run. os.Signal never
// crosses into internal/ — deliveries are adapted to pure struct{} pulses
// here (#700).
func registerForceExit(exitFn func(int), sigs ...os.Signal) {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, sigs...)

	pulses := make(chan struct{}, signalChannelBuffer)

	go forwardAsPulses(sigCh, pulses)

	cli.ForceExitOnRepeatedSignal(pulses, exitFn)
}
