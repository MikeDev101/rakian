package keypad

import "context"

type KeypadEvent struct {
	State    bool
	Key      rune
	Duration float64
}

var KeyMap = map[[2]int]rune{
	{0, 1}: 'C',
	{0, 2}: '1',
	{0, 3}: '2',
	{0, 4}: '3',
	{1, 1}: 'S',
	{1, 2}: '4',
	{1, 3}: '5',
	{1, 4}: '6',
	{2, 0}: 'D',
	{2, 2}: '7',
	{2, 3}: '8',
	{2, 4}: '9',
	{3, 1}: 'U',
	{3, 2}: '*',
	{3, 3}: '0',
	{3, 4}: '#',
}

type Keypad interface {
	Run(ctx context.Context, debug bool) <-chan *KeypadEvent
}
