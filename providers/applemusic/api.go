package applemusic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dgrinbergs/universal-music-library/music"
)

const apiBase = "https://api.music.apple.com/v1"

// Apple Music API response types

type appleMusicResponse[T any] struct {
	Data []T    `json:"data"`
	Next string `json:"next"`
}

type searchResponse struct {
	Results struct {
		Songs appleMusicResponse[appleMusicSong] `json:"songs"`
	} `json:"results"`
}

type appleMusicSong struct {
	ID         string              `json:"id"`
	Type       string              `json:"type"`
	Attributes appleMusicSongAttrs `json:"attributes"`
}

type appleMusicSongAttrs struct {
	Name             string `json:"name"`
	ArtistName       string `json:"artistName"`
	AlbumName        string `json:"albumName"`
	DurationInMillis int    `json:"durationInMillis"`
	ISRC             string `json:"isrc"`
}

type appleMusicPlaylist struct {
	ID         string                      `json:"id"`
	Type       string                      `json:"type"`
	Attributes appleMusicPlaylistAttrs     `json:"attributes"`
}

type appleMusicPlaylistAttrs struct {
	Name        string `json:"name"`
	Description *struct {
		Standard string `json:"standard"`
	} `json:"description,omitempty"`
}

type createPlaylistBody struct {
	Attributes struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	} `json:"attributes"`
}

type trackRelationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type addTracksBody struct {
	Data []trackRelationshipData `json:"data"`
}

func (s appleMusicSong) toTrack() music.Track {
	return music.Track{
		Title:    s.Attributes.Name,
		Artist:   s.Attributes.ArtistName,
		Album:    s.Attributes.AlbumName,
		ISRC:     s.Attributes.ISRC,
		Duration: time.Duration(s.Attributes.DurationInMillis) * time.Millisecond,
	}
}

// HTTP helpers

func doRequest(method, fullURL string, body io.Reader) (*http.Response, error) {
	devToken, err := getDeveloperToken()
	if err != nil {
		return nil, err
	}
	userToken, err := getMusicUserToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+devToken)
	req.Header.Set("Music-User-Token", userToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

func apiGet[T any](fullURL string) (*T, error) {
	resp, err := doRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("apple music API error %d: %s", resp.StatusCode, respBody)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Search

func searchByISRC(storefront, isrc string) (*appleMusicSong, error) {
	u := fmt.Sprintf("%s/catalog/%s/songs?filter[isrc]=%s", apiBase, storefront, url.QueryEscape(isrc))
	result, err := apiGet[appleMusicResponse[appleMusicSong]](u)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no results for ISRC %s", isrc)
	}
	return &result.Data[0], nil
}

func searchByTerm(storefront string, track music.Track) (*appleMusicSong, error) {
	term := fmt.Sprintf("%s %s", track.Title, track.Artist)
	u := fmt.Sprintf("%s/catalog/%s/search?types=songs&limit=1&term=%s", apiBase, storefront, url.QueryEscape(term))
	result, err := apiGet[searchResponse](u)
	if err != nil {
		return nil, err
	}
	if len(result.Results.Songs.Data) == 0 {
		return nil, fmt.Errorf("track not found: %s — %s", track.Artist, track.Title)
	}
	return &result.Results.Songs.Data[0], nil
}

func findTrack(storefront string, track music.Track) (*appleMusicSong, error) {
	if track.ISRC != "" {
		song, err := searchByISRC(storefront, track.ISRC)
		if err == nil {
			return song, nil
		}
	}
	return searchByTerm(storefront, track)
}

// Library operations

func getAllPlaylists() ([]appleMusicPlaylist, error) {
	var all []appleMusicPlaylist
	nextURL := apiBase + "/me/library/playlists?limit=25"

	for nextURL != "" {
		page, err := apiGet[appleMusicResponse[appleMusicPlaylist]](nextURL)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.Next == "" {
			break
		}
		nextURL = apiBase + page.Next
	}
	return all, nil
}

func createPlaylist(name string) (*appleMusicPlaylist, error) {
	body := createPlaylistBody{}
	body.Attributes.Name = name

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := doRequest("POST", apiBase+"/me/library/playlists", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("apple music API error %d: %s", resp.StatusCode, respBody)
	}

	var result appleMusicResponse[appleMusicPlaylist]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no playlist returned in response")
	}
	return &result.Data[0], nil
}

func addTracksToPlaylist(playlistID string, songIDs []string) error {
	// Apple Music allows ~25 tracks per request
	for i := 0; i < len(songIDs); i += 25 {
		end := min(i+25, len(songIDs))
		items := make([]trackRelationshipData, 0, end-i)
		for _, id := range songIDs[i:end] {
			items = append(items, trackRelationshipData{ID: id, Type: "songs"})
		}

		data, err := json.Marshal(addTracksBody{Data: items})
		if err != nil {
			return err
		}

		resp, err := doRequest("POST",
			fmt.Sprintf("%s/me/library/playlists/%s/tracks", apiBase, playlistID),
			bytes.NewReader(data))
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("apple music API error %d adding tracks", resp.StatusCode)
		}
	}
	return nil
}

func addSongsToLibrary(songIDs []string) error {
	// Apple Music allows adding songs to library via query params
	for i := 0; i < len(songIDs); i += 25 {
		end := min(i+25, len(songIDs))
		ids := strings.Join(songIDs[i:end], ",")

		resp, err := doRequest("POST",
			fmt.Sprintf("%s/me/library?ids[songs]=%s", apiBase, ids),
			nil)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("apple music API error %d adding songs to library", resp.StatusCode)
		}
	}
	return nil
}
