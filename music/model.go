package music

import "time"

type Track struct {
	Title    string        `json:"title" yaml:"title"`
	Artist   string        `json:"artist" yaml:"artist"`
	Album    string        `json:"album" yaml:"album"`
	ISRC     string        `json:"isrc,omitempty" yaml:"isrc,omitempty"`
	Duration time.Duration `json:"duration" yaml:"duration"`
}

type Playlist struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Tracks      []Track `json:"tracks" yaml:"tracks"`
}

type Library struct {
	SavedTracks []Track    `json:"saved_tracks" yaml:"saved_tracks"`
	Playlists   []Playlist `json:"playlists" yaml:"playlists"`
}
