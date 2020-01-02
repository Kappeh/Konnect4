package main

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	// Player1 represents either player1's turn, player1's tile
	// or that player1 is the winner.
	Player1 = iota
	// Player2 represents either player2's turn, player2's tile
	// or that player2 is the winner.
	Player2
	// Empty represents an empty tile or that no one has won yet.
	Empty
	// Tie represents that neither player won.
	Tie
)

// State represents a position in a connect 4 game.
// This includes the positions of any placed tiles, the current
// player and the winner, if there is one.
type State struct {
	// Tiles is all of the cells on the connect 4 board.
	// Each cell can be Empty, Player1 or Player2.
	Tiles [42]int
	// Player is the current player. Either Player1 or Player2.
	Player int
	// Winner can be Empty, in this case the game is not over.
	// Otherwise, it's Player1, Player2 or Tie.
	Winner int
}

// StateFromCFP will generate a State object from a string
// that is in line with the CFP position reperesentation.
func StateFromCFP(p string) (State, error) {
	result := State{}
	if len(p) != 43 {
		return result, errors.New("invalid position")
	}
	for i, v := range p[:42] {
		switch v {
		case '0':
			result.Tiles[i] = Empty
		case '1':
			result.Tiles[i] = Player1
		case '2':
			result.Tiles[i] = Player2
		default:
			return result, errors.New("invalid position")
		}
	}
	switch p[42] {
	case '1':
		result.Player = Player1
	case '2':
		result.Player = Player2
	default:
		return result, errors.New("invalid position")
	}
	result.Winner = result.calculateWinner()
	return result, nil
}

// NewState returns a State that represents a new game position.
func NewState() State {
	result := State{
		Player: Player1,
		Winner: Empty,
	}
	for i := 0; i < 42; i++ {
		result.Tiles[i] = Empty
	}
	return result
}

// LegalActions produces a one-hot array of which moves are legal.
func (s State) LegalActions() [7]bool {
	if s.Winner != Empty {
		return [7]bool{}
	}
	result := [7]bool{}
	for i := 0; i < 7; i++ {
		result[i] = s.Tiles[i] == Empty
	}
	return result
}

func (s State) dropTile(player, column int) (State, error) {
	i := column
	for i < 42 && s.Tiles[i] == Empty {
		i += 7
	}
	i -= 7
	if i < 0 {
		return s, errors.New("illegal move")
	}
	s.Tiles[i] = player
	return s, nil
}

// NextState updates the state as if a player dropped a tile
// into the board.
func (s State) NextState(column int) (State, error) {
	// Update the tiles in state
	result, err := s.dropTile(s.Player, column)
	if err != nil {
		return s, errors.Wrap(err, "couldn't perform action")
	}
	// Update winner
	winningMove, err := result.isWinningMove(column)
	if err != nil {
		return result, errors.Wrap(err, "failed to perform win check")
	}
	if winningMove {
		result.Winner = result.Player
	}
	// Switch players
	if result.Player == Player1 {
		result.Player = Player2
	} else {
		result.Player = Player1
	}
	return result, nil
}

// Each set of 4 values refer to a possible 4 in a row.
var lines = [...]int{
	// Horizontal lines
	0, 1, 2, 3, 1, 2, 3, 4, 2, 3, 4, 5, 3, 4, 5, 6,
	7, 8, 9, 10, 8, 9, 10, 11, 9, 10, 11, 12, 10, 11, 12, 13,
	14, 15, 16, 17, 15, 16, 17, 18, 16, 17, 18, 19, 17, 18, 19, 20,
	21, 22, 23, 24, 22, 23, 24, 25, 23, 24, 25, 26, 24, 25, 26, 27,
	28, 29, 30, 31, 29, 30, 31, 32, 30, 31, 32, 33, 31, 32, 33, 34,
	35, 36, 37, 38, 36, 37, 38, 39, 37, 38, 39, 40, 38, 39, 40, 41,
	// Vertical lines
	0, 7, 14, 21, 7, 14, 21, 28, 14, 21, 28, 35,
	1, 8, 15, 22, 8, 15, 22, 29, 15, 22, 29, 36,
	2, 9, 16, 23, 9, 16, 23, 30, 16, 23, 30, 37,
	3, 10, 17, 24, 10, 17, 24, 31, 17, 24, 31, 38,
	4, 11, 18, 25, 11, 18, 25, 32, 18, 25, 32, 39,
	5, 12, 19, 26, 12, 19, 26, 33, 19, 26, 33, 40,
	6, 13, 20, 27, 13, 20, 27, 34, 20, 27, 34, 41,
	// Positive diagonals
	0, 8, 16, 24, 1, 9, 18, 26, 2, 10, 18, 26, 3, 11, 19, 27,
	7, 15, 23, 31, 8, 16, 24, 32, 9, 17, 25, 33, 10, 18, 26, 11,
	14, 22, 30, 38, 15, 23, 31, 39, 16, 24, 32, 40, 17, 25, 33, 41,
	// Negative diagonals
	3, 9, 15, 21, 4, 10, 16, 22, 5, 11, 17, 23, 6, 12, 18, 24,
	10, 16, 22, 28, 11, 17, 23, 29, 12, 18, 24, 30, 13, 19, 25, 31,
	17, 23, 29, 35, 18, 24, 30, 36, 19, 25, 31, 37, 20, 26, 32, 38,
}

// calculateWinner assumes that there is only one player
// that has a four in a row. It will return the player
// of the first four in a row it finds.
// As this checks every possible four in a row, it's advised
// to avoid using it.
func (s State) calculateWinner() int {
	// Check each possible 4 in a row
LINE_LOOP:
	for i := 0; i < len(lines); i += 4 {
		player := s.Tiles[lines[i]]
		if player == Empty {
			continue LINE_LOOP
		}
		for j := 1; j < 4; j++ {
			index := i + j
			if s.Tiles[lines[index]] != player {
				continue LINE_LOOP
			}
		}
		return player
	}
	// If there is no four in a row,
	// check if the board is not full
	for i := 0; i < 42; i++ {
		if s.Tiles[i] == Empty {
			return Empty
		}
	}
	// If the board is full, it's a tie
	return Tie
}

func (s State) checkForFour(player, x, y, dx, dy int) (bool, error) {
	if x < 0 || x >= 7 || y < 0 || y >= 6 {
		return false, errors.New("index out or range")
	}
	var (
		count  = 1
		cx     int
		cy     int
		index  int
		dindex int
	)
	cx = x + dx
	cy = y + dy
	index = cx + 7*cy
	dindex = dx + 7*dy
	for cx >= 0 && cx < 7 && cy >= 0 && cy < 6 {
		cx += dx
		cy += dy
		if s.Tiles[index] != player {
			break
		}
		index += dindex
		count++
	}
	cx = x - dx
	cy = y - dy
	index = cx + 7*cy
	dindex = -dindex
	for cx >= 0 && cx < 7 && cy >= 0 && cy < 6 {
		cx -= dx
		cy -= dy
		if s.Tiles[index] != player {
			break
		}
		index += dindex
		count++
	}
	return count >= 4, nil
}

func (s State) checkAllDirections(player, x, y int) (bool, error) {
	var (
		win bool
		err error
	)
	// Horizontal line
	win, err = s.checkForFour(player, x, y, 1, 0)
	if err != nil {
		return false, errors.Wrap(err, "failed to check direction")
	} else if win {
		return true, nil
	}
	// Vertical line
	win, err = s.checkForFour(player, x, y, 0, 1)
	if err != nil {
		return false, errors.Wrap(err, "failed to check direction")
	} else if win {
		return true, nil
	}
	// Positive diagonal line
	win, err = s.checkForFour(player, x, y, 1, 1)
	if err != nil {
		return false, errors.Wrap(err, "failed to check direction")
	} else if win {
		return true, nil
	}
	// Negative diagonal line
	win, err = s.checkForFour(player, x, y, 1, -1)
	if err != nil {
		return false, errors.Wrap(err, "failed to check direction")
	}
	return win, nil
}

// Call DIRECTLY AFTER the tiles have been updated for the move
// This is used to reduce the amount of computation spent on
// checking for fours in a row.
// This will only check the rows that include the last piece dropped.
func (s State) isWinningMove(column int) (bool, error) {
	if column < 0 || column >= 7 {
		return false, errors.New("index out of range")
	}
	index := column
	for index < 42-7 && s.Tiles[index] == Empty {
		index += 7
	}
	player := s.Tiles[index]
	if player == Empty {
		return false, errors.New("move wasn't taken")
	}
	return s.checkAllDirections(player, index%7, index/7)
}

func (s State) String() string {
	lines := [6]string{}
	for row := 0; row < 6; row++ {
		cells := [7]string{}
		for cell := 0; cell < 7; cell++ {
			index := cell + 7*row
			switch s.Tiles[index] {
			case Player1:
				cells[cell] = "X"
			case Player2:
				cells[cell] = "O"
			case Empty:
				cells[cell] = "-"
			}
		}
		lines[row] = strings.Join(cells[:], " ")
	}
	return strings.Join(lines[:], "\n")
}

// CFPString returns a string that represents the state
// in compliance with the CFP position representation.
func (s State) CFPString() string {
	result := [43]rune{}
	for i := 0; i < 42; i++ {
		switch s.Tiles[i] {
		case Player1:
			result[i] = '1'
		case Player2:
			result[i] = '2'
		case Empty:
			result[i] = '0'
		}
	}
	switch s.Player {
	case Player1:
		result[42] = '1'
	case Player2:
		result[42] = '2'
	}
	return string(result[:])
}
