package tracks

import (
	"fmt"
	"strings"
)

const (
	KeyUnknown uint = iota
	KeyA
	KeyASharp
	KeyB
	KeyC
	KeyCSharp
	KeyD
	KeyDSharp
	KeyE
	KeyF
	KeyFSharp
	KeyG
	KeyGSharp
)

const (
	ModeUnknown uint = iota
	ModeMinor
	ModeMajor
)

const (
	KeyNameUnknown = "unknown"
	KeyNameA       = "A"
	KeyNameASharp  = "A#"
	KeyNameB       = "B"
	KeyNameC       = "C"
	KeyNameCSharp  = "C#"
	KeyNameD       = "D"
	KeyNameDSharp  = "D#"
	KeyNameE       = "E"
	KeyNameF       = "F"
	KeyNameFSharp  = "F#"
	KeyNameG       = "G"
	KeyNameGSharp  = "G#"
)

var KeysMapping = map[uint]string{
	KeyUnknown: KeyNameUnknown,
	KeyA:       KeyNameA,
	KeyASharp:  KeyNameASharp,
	KeyB:       KeyNameB,
	KeyC:       KeyNameC,
	KeyCSharp:  KeyNameCSharp,
	KeyD:       KeyNameD,
	KeyDSharp:  KeyNameDSharp,
	KeyE:       KeyNameE,
	KeyF:       KeyNameF,
	KeyFSharp:  KeyNameFSharp,
	KeyG:       KeyNameG,
	KeyGSharp:  KeyNameGSharp,
}

const (
	ModeNameMinor = "minor"
	ModeNameMajor = "major"
)

var ModesMapping = map[uint]string{
	ModeMinor: ModeNameMinor,
	ModeMajor: ModeNameMajor,
}

type Track struct {
	Model
	FilePath string `json:"-"`

	Title            string `json:"title"`
	Artist           string `json:"artist"`
	Album            string `json:"album"`
	AlbumTrackNumber uint   `json:"album_track_number"`
	Tags             []Tag  `json:"tags,omitempty" gorm:"many2many:track_tags;"`
	Played           uint64 `json:"played"`

	AuthorID uint64  `json:"author_id,omitempty"`
	Author   *Author `json:"author,omitempty"`

	// JamPlayer info
	Length    uint64 `json:"length"`
	LoopStart uint64 `json:"loop_start"`
	LoopEnd   uint64 `json:"loop_end"`
	BPM       uint   `json:"bpm"`
	BPI       uint   `json:"bpi"`
	Key       uint   `json:"key"`
	Mode      uint   `json:"mode"`

	// Loudness
	Integrated float32 `json:"integrated"`
	Range      float32 `json:"range"`
	Peak       float32 `json:"peak"`
	Shortterm  float32 `json:"shortterm"`
	Momentary  float32 `json:"momentary"`
}

type Tag struct {
	Model
	Name string `json:"name"`
}

func (t Track) KeyString() string {
	return KeysMapping[t.Key] + " " + ModesMapping[t.Mode]
}

func (t Track) String() string {
	trackName := fmt.Sprintf("%s (%s, %d BPM)", t.Title, t.KeyString(), t.BPM)

	if t.Artist != "" {
		trackName += fmt.Sprintf(" by %s", t.Artist)
	}
	if t.Album != "" {
		trackName += fmt.Sprintf(" (%s)", t.Album)
	}

	return trackName
}

func (p *Track) AfterFind() (err error) {

	p.Artist = strings.Trim(p.Artist, fmt.Sprintf("\x00 \n"))
	p.Album = strings.Trim(p.Album, fmt.Sprintf("\x00 \n"))
	p.Title = strings.Trim(p.Title, fmt.Sprintf("\x00 \n"))

	return
}
