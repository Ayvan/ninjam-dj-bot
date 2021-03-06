package lib

import (
	"fmt"
	"github.com/ayvan/ninjam-dj-bot/tracks"
	"strings"
)

type KeyMode struct {
	Key  uint
	Mode uint
}

var keysAliases = map[uint][]string{
	tracks.KeyA:      {"A"},
	tracks.KeyASharp: {"A#", "Bb"},
	tracks.KeyB:      {"B"},
	tracks.KeyC:      {"C"},
	tracks.KeyCSharp: {"C#", "Db"},
	tracks.KeyD:      {"D"},
	tracks.KeyDSharp: {"D#", "Eb"},
	tracks.KeyE:      {"E", "Fb"},
	tracks.KeyF:      {"F", "E#"},
	tracks.KeyFSharp: {"F#", "Gb"},
	tracks.KeyG:      {"G"},
	tracks.KeyGSharp: {"G#", "Ab"},
}

var modesAliases = map[uint][]string{
	tracks.ModeMinor: {"m", "minor"},
	tracks.ModeMajor: {"", "major"},
}

var keysMap = make(map[string]KeyMode)

func init() {
	for mode, mAliases := range modesAliases {
		for _, modeName := range mAliases {
			for key, kAliases := range keysAliases {
				for _, keyName := range kAliases {
					name := fmt.Sprintf("%s%s", keyName, modeName)
					name = strings.ToLower(name)
					keysMap[name] = KeyMode{Mode: mode, Key: key}
				}
			}
		}
	}
}

func KeyModeByName(name string) KeyMode {
	return keysMap[strings.ToLower(name)]
}
