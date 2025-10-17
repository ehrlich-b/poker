package cards

import (
	"fmt"
	"strings"
)

// Rank represents a card rank (2-A)
type Rank uint8

const (
	Two Rank = iota
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
	Ace
)

// Suit represents a card suit
type Suit uint8

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

// Card represents a single playing card
type Card struct {
	Rank Rank
	Suit Suit
}

// NewCard creates a card from rank and suit
func NewCard(rank Rank, suit Suit) Card {
	return Card{Rank: rank, Suit: suit}
}

// ParseCard parses a card from string notation (e.g., "As", "Kh", "Td")
func ParseCard(s string) (Card, error) {
	s = strings.TrimSpace(s)
	if len(s) != 2 {
		return Card{}, fmt.Errorf("invalid card string: %q (must be 2 characters)", s)
	}

	rank, err := parseRank(s[0])
	if err != nil {
		return Card{}, err
	}

	suit, err := parseSuit(s[1])
	if err != nil {
		return Card{}, err
	}

	return Card{Rank: rank, Suit: suit}, nil
}

// parseRank converts a character to a Rank
func parseRank(b byte) (Rank, error) {
	switch b {
	case '2':
		return Two, nil
	case '3':
		return Three, nil
	case '4':
		return Four, nil
	case '5':
		return Five, nil
	case '6':
		return Six, nil
	case '7':
		return Seven, nil
	case '8':
		return Eight, nil
	case '9':
		return Nine, nil
	case 'T', 't':
		return Ten, nil
	case 'J', 'j':
		return Jack, nil
	case 'Q', 'q':
		return Queen, nil
	case 'K', 'k':
		return King, nil
	case 'A', 'a':
		return Ace, nil
	default:
		return 0, fmt.Errorf("invalid rank: %c", b)
	}
}

// parseSuit converts a character to a Suit
func parseSuit(b byte) (Suit, error) {
	switch b {
	case 's', 'S':
		return Spades, nil
	case 'h', 'H':
		return Hearts, nil
	case 'd', 'D':
		return Diamonds, nil
	case 'c', 'C':
		return Clubs, nil
	default:
		return 0, fmt.Errorf("invalid suit: %c", b)
	}
}

// String returns the card in standard notation (e.g., "As", "Kh")
func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Rank, c.Suit)
}

// String returns the rank as a single character
func (r Rank) String() string {
	switch r {
	case Two:
		return "2"
	case Three:
		return "3"
	case Four:
		return "4"
	case Five:
		return "5"
	case Six:
		return "6"
	case Seven:
		return "7"
	case Eight:
		return "8"
	case Nine:
		return "9"
	case Ten:
		return "T"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	case Ace:
		return "A"
	default:
		return "?"
	}
}

// String returns the suit as a single character
func (s Suit) String() string {
	switch s {
	case Spades:
		return "s"
	case Hearts:
		return "h"
	case Diamonds:
		return "d"
	case Clubs:
		return "c"
	default:
		return "?"
	}
}

// ParseCards parses multiple cards from a string (e.g., "AsKhQd")
func ParseCards(s string) ([]Card, error) {
	s = strings.ReplaceAll(s, " ", "")
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("invalid cards string: %q (must have even length)", s)
	}

	cards := make([]Card, 0, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		card, err := ParseCard(s[i : i+2])
		if err != nil {
			return nil, fmt.Errorf("error parsing card at position %d: %w", i, err)
		}
		cards = append(cards, card)
	}

	return cards, nil
}
