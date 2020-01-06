package main

// Communicator is a way for the gui to communicate with the engine
// Each implimentation would be a different protocol
type Communicator interface {
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
	Go(float32) error
	// Stop tells the engine to stop thinking as soon as possible
	// The best move the engine found is returned
	Stop() (int, error)
	// Quit should close all connections to the process. and
	// tell the engine to quit as soon as possible.
	Quit() error
	// InfoChannel should return a channel which get's populated
	// with information outputted by the engine
	InfoChannel() <-chan string
}
