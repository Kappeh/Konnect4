package main

import (
	"time"

	"github.com/pkg/errors"
)

const (
	// DefaultTurnTime is the default amount of time
	// that a player will be given to analyse a position
	// before being asked to provide a move
	DefaultTurnTime = 5 * time.Second
)

// Game is an environment for two players to play a game of
// connect 4. It includes methods to control the game and
// the game loop.
type Game struct {
	// Player1Status is the index in History of the position
	// Player1 has in its internal state
	// -1 means that the player needs to be told about a new game
	Player1Status int
	Player1       *Engine
	// Player2Status is the index in History of the position
	// Player2 has in its internal state
	// -1 means that the player needs to be told about a new game
	Player2Status int
	Player2       *Engine

	// TurnTime is the amount of time a player will be given
	// to analyse a position before being asked to provide
	// a move
	TurnTime time.Duration

	// State is the current state of the board
	State State
	// History is the positions that have been visited over
	// the course of the game
	History [42]State
	// HistoryIndex is the index of the current state in History
	HistoryIndex int

	// Running tracks whether the gameloop is running or not
	Running bool
	// PauseSignal is for sending a signal into the gameloop
	// from another goroutine to stop
	PauseSignal chan bool

	// Events is where all events that happen when the game is
	// running is to be sent. This includes when a new position
	// is reached and when the game is over
	Events chan<- GameEvent
}

// GameEvent is an interface that allows multiple types of events
// to be handled using the same channel
type GameEvent interface {
	GameEvent()
}

// NewStateEvent is triggered when a new position is reached
type NewStateEvent struct {
	State State
}

// GameEvent allows NewStateEvent to impliment the GameEvent interface
func (NewStateEvent) GameEvent() {}

// GameOverEvent is triggered when the game finishes
type GameOverEvent struct {
	Winner int
}

// GameEvent allows GameOverEvent to impliment the GameEvent interface
func (GameOverEvent) GameEvent() {}

// ErrorEvent is triggered when an error occurs when playing game
type ErrorEvent struct {
	Error error
}

// GameEvent allows ErrorEvent to impliment the GameEvent interface
func (ErrorEvent) GameEvent() {}

// NewGame returns a new game with the default timeout options
// and a new starting position
func NewGame() *Game {
	return &Game{
		TurnTime:    DefaultTurnTime,
		State:       NewState(),
		History:     [42]State{NewState()},
		PauseSignal: make(chan bool),
	}
}

// SetPlayer1 sets the first player of the game to a provided engine
func (g *Game) SetPlayer1(e *Engine) error {
	// Return an error if the game is currently running
	if g.Running {
		return errors.New("cannot set player while game is being played")
	}
	// Return an error if the engine provided is nil
	if e == nil {
		return errors.New("player1 cannot be nil")
	}
	// Set the player
	g.Player1 = e
	if e == g.Player2 {
		// If the two players are the same, they have the same
		// internal state
		g.Player1Status = g.Player2Status
	} else {
		// Otherwise, the internal state hasn't seen any board yet
		g.Player1Status = -1
	}
	return nil
}

// SetPlayer2 sets the first player of the game to a provided engine
func (g *Game) SetPlayer2(e *Engine) error {
	// Return an error if the game is currently running
	if g.Running {
		return errors.New("cannot set player while game is being played")
	}
	// Return an error if the engine provided is nil
	if e == nil {
		return errors.New("player2 cannot be nil")
	}
	// Set the player
	g.Player2 = e
	if e == g.Player1 {
		// If the two players are the same, they have the same
		// internal state
		g.Player2Status = g.Player1Status
	} else {
		// Otherwise, the internal state hasn't seen any board yet
		g.Player2Status = -1
	}
	return nil
}

// SetTimeout sets the time that the players will be provided to analyse
// the board before being asked to provide a move
func (g *Game) SetTimeout(time time.Duration) error {
	// Return an error if the game is running
	if g.Running {
		return errors.New("cannot set timout while game is being played")
	}
	// Return an error if the time is not positive
	if time <= 0 {
		return errors.New("time must be positive")
	}
	// Set the time
	g.TurnTime = time
	return nil
}

// Reset sets the game back to a starting position
func (g *Game) Reset() error {
	// Return an error if the game is running
	if g.Running {
		return errors.New("cannot reset while game is being played")
	}
	// Set the position to a starting board
	return g.Position(NewState())
}

// Position sets the game to a provided state
func (g *Game) Position(s State) error {
	// Return an error if the game is running
	if g.Running {
		return errors.New("cannot set new position while game is being played")
	}
	// Set up the game state
	g.State = s
	g.History = [42]State{s}
	g.HistoryIndex = 0
	g.Player1Status = -1
	g.Player2Status = -1
	return nil
}

// Play runs the game to completion, using player1 and player2 to
// provide moves in each board state
func (g *Game) Play() error {
	// Return an error if the game is already running
	if g.Running {
		return errors.New("game is already being played")
	}
	// Return error if either player1 or player2 is nil
	if g.Player1 == nil || g.Player2 == nil {
		return errors.New("cannot play game when player is nil")
	}
	// Set the running state of the game
	g.Running = true
	// Start gameloop
	go g.gameLoop()
	// Game has finished being played, return successfully
	return nil
}

// Pause sends a pause signal to the gameloop internals which stops it
func (g *Game) Pause() error {
	// Return an error if the game isn't already running
	if !g.Running {
		return errors.New("game is not being played")
	}
	// Update the running state and send pause signal to gameloop
	g.Running = false
	g.PauseSignal <- true
	return nil
}

// NotifyEvents sets the channel in which game events
// are to be sent to. This includes when the game is over
// and when a new position is reached
func (g *Game) NotifyEvents(channel chan<- GameEvent) {
	g.Events = channel
}

// currentPlayer gets the player that is to make the next move
func (g *Game) currentPlayer() (*Engine, error) {
	var player *Engine
	if g.State.Player == Player1 {
		player = g.Player1
	} else {
		player = g.Player2
	}
	if player == nil {
		// Return an error if the player is nil
		return nil, errors.New("current player is nil")
	}
	return player, nil
}

func (g *Game) gameLoop() {
	// Loop until the game is finished or the running state changes
	for g.State.Winner == Empty && g.Running {
		// Play out a turn and return errors if they arise
		err := g.playTurn()
		if err != nil && g.Events != nil {
			g.Events <- ErrorEvent{
				Error: errors.Wrap(err, "couldn't play turn"),
			}
		}
		if err != nil {
			g.Running = false
			return
		}
		if g.Events != nil {
			g.Events <- NewStateEvent{State: g.State}
		}
	}
	if g.Events != nil {
		g.Events <- GameOverEvent{Winner: g.State.Winner}
	}
	g.Running = false
}

// playTurn plays out the next turn of the game
func (g *Game) playTurn() error {
	// Return an error if the game is over
	if g.State.Winner != Empty {
		return errors.New("unable to play turn when game is over")
	}
	// Update the engines' internal states
	err := g.updateEngineStates()
	if err != nil {
		return errors.Wrap(err, "couldn't update engine states")
	}
	// Get the player that is to make the next move
	player, err := g.currentPlayer()
	if err != nil {
		return errors.Wrap(err, "couldn't get current player")
	}
	// Get the player to analyse the current position
	err = player.Go(g.TurnTime)
	if err != nil {
		return errors.Wrap(err, "failed to start player analysis")
	}
	// Wait for a pause signal or the timeout to pass
	select {
	case <-time.After(g.TurnTime):
	case <-g.PauseSignal:
		// If a pause signal is sent, stop the play from thinking
		_, err := player.Stop()
		if err != nil {
			return errors.Wrap(err, "unable to send stop signal to player")
		}
		return nil
	}
	// Get the player from the player
	move, err := player.Stop()
	if err != nil {
		return errors.Wrap(err, "unable to get move from player")
	}
	// Apply the move to the current state
	g.State, err = g.State.NextState(move)
	if err != nil {
		return errors.Wrap(err, "unable to apply move")
	}
	// Update the history of the game
	g.HistoryIndex++
	g.History[g.HistoryIndex] = g.State
	return nil
}

// updateEngineStatuses sends relevent information to the players
// to keep their internal state in sync with the current game state
func (g *Game) updateEngineStates() error {
	// Update the first players state
	err := g.updateEngineState(g.Player1, g.Player1Status)
	if err != nil {
		return errors.Wrap(err, "couldn't update player1 state")
	}
	// Update the player1 status
	g.Player1Status = g.HistoryIndex
	// If the players are the same
	if g.Player1 == g.Player2 {
		// Player2 is already up to date
		g.Player2Status = g.HistoryIndex
		return nil
	}
	// Otherwise, update the seconds players state
	err = g.updateEngineState(g.Player2, g.Player2Status)
	if err != nil {
		return errors.Wrap(err, "couldn't update player2 state")
	}
	// Update the player2 status
	g.Player2Status = g.HistoryIndex
	return nil
}

// updateEngineState send relevent information to a player
// to keep their internal state in sync with the current game state
func (g *Game) updateEngineState(e *Engine, status int) error {
	// Return an error if the player is nil
	if e == nil {
		return errors.New("engine is nil")
	}
	// If the status is -1, then send a newgame signal to the player
	if status == -1 {
		err := e.NewGame()
		if err != nil {
			return errors.Wrap(err, "couldn't send engine newgame signal")
		}
	}
	// If the status shows the players internal state isn't up to date,
	// send the current position to the player
	if status < g.HistoryIndex {
		err := e.Position(g.State)
		if err != nil {
			return errors.Wrap(err, "couldn't send engine position signal")
		}
	}
	return nil
}
