package main

import (
	"time"
)

// Protocol is a way for the gui to communicate with the engine
// Each implimentation would be a different protocol
type Protocol interface {
	// Handshake connects to a process and performs a protocol
	// handshake. The name, author and options should be
	// aquired from the engine during this process
	Handshake(*string, *string, *map[string]Option) error
	// Debug enables or disables debug mode on the engine depending
	// on the bool parameter passed into it.
	// true = enable debug, false = disable debug
	Debug(bool) error
	// SetOption trys to set an internal parameter of the
	// engine.
	SetOption(Option) error
	// NewGame should tell the engine that the next position
	// is from a different game to the previous position.
	NewGame() error
	// Position sends a new position for the engine to analyse
	// If the position is from a new game, this will be
	// preceeded by a call to NewGame()
	Position(State) error
	// Go tells the engine that it should start analysing the
	// position and the maximum amount of time it has to think
	Go(time.Duration) error
	// Stop tells the engine to stop thinking as soon as possible
	// The best move the engine found is returned
	Stop() (int, error)
	// Quit should close all connections to the process. and
	// tell the engine to quit as soon as possible.
	Quit() error
	// NotifyInfo tells the protocol to send any info events to
	// the provided channel
	NotifyInfo(chan<- string)
	// NotifyComm tells the protocol to send any communications
	// between the protocol implimentation and the actial engine
	// to the provided channel.
	NotifyComm(chan<- Communication)
}

// Communication is a message that has been send either to the engine
// or received from the engine
type Communication struct {
	// Time is the time that the communication happened
	Time time.Time
	// ToEngine indicated the direction of the message
	// e.g. true = to the engine, false = from the engine
	ToEngine bool
	// Message is the actual command that was sent.
	Message string
}
