package spotify

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

const apiBase = "https://api.spotify.com/v1"

// Spotify API response types

type paginatedResponse[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next"`
	Total int    `json:"total"`
}

type spotifyPlaylist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tracks      struct {
		Total int `json:"total"`
	} `json:"tracks"`
}

type spotifyPlaylistTrackItem struct {
	Track *spotifyTrack `json:"track"`
}

type spotifySavedTrackItem struct {
	Track spotifyTrack `json:"track"`
}

type spotifyTrack struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Artists     []spotifyArtist   `json:"artists"`
	Album       spotifyAlbum      `json:"album"`
	DurationMS  int               `json:"duration_ms"`
	ExternalIDs map[string]string `json:"external_ids"`
	URI         string            `json:"uri"`
}

type spotifyArtist struct {
	Name string `json:"name"`
}

type spotifyAlbum struct {
	Name string `json:"name"`
}

type spotifyUser struct {
	ID string `json:"id"`
}

type spotifySearchResult struct {
	Tracks paginatedResponse[spotifyTrack] `json:"tracks"`
}

type createPlaylistRequest struct {
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

type addTracksRequest struct {
	URIs []string `json:"uris"`
}

func (t spotifyTrack) toTrack() music.Track {
	artists := make([]string, len(t.Artists))
	for i, a := range t.Artists {
		artists[i] = a.Name
	}
	return music.Track{
		Title:    t.Name,
		Artist:   strings.Join(artists, ", "),
		Album:    t.Album.Name,
		ISRC:     t.ExternalIDs["isrc"],
		Duration: time.Duration(t.DurationMS) * time.Millisecond,
	}
}

// HTTP helpers

func apiGet[T any](fullURL string) (*T, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, body)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func apiPost[T any](fullURL string, body any) (*T, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, respBody)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Paginated fetchers

func getAllPlaylists() ([]spotifyPlaylist, error) {
	var all []spotifyPlaylist
	nextURL := apiBase + "/me/playlists?limit=50"

	for nextURL != "" {
		page, err := apiGet[paginatedResponse[spotifyPlaylist]](nextURL)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		nextURL = page.Next
	}
	return all, nil
}

func getPlaylistTracks(playlistID string) ([]spotifyTrack, error) {
	var all []spotifyTrack
	nextURL := fmt.Sprintf("%s/playlists/%s/tracks?limit=50", apiBase, playlistID)

	for nextURL != "" {
		page, err := apiGet[paginatedResponse[spotifyPlaylistTrackItem]](nextURL)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if item.Track != nil {
				all = append(all, *item.Track)
			}
		}
		nextURL = page.Next
	}
	return all, nil
}

func getAllSavedTracks() ([]spotifyTrack, error) {
	var all []spotifyTrack
	nextURL := apiBase + "/me/tracks?limit=50"

	for nextURL != "" {
		page, err := apiGet[paginatedResponse[spotifySavedTrackItem]](nextURL)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			all = append(all, item.Track)
		}
		nextURL = page.Next
	}
	return all, nil
}

func searchTrack(track music.Track) (*spotifyTrack, error) {
	// Try ISRC first for exact matching
	if track.ISRC != "" {
		q := url.QueryEscape(fmt.Sprintf("isrc:%s", track.ISRC))
		result, err := apiGet[spotifySearchResult](fmt.Sprintf("%s/search?q=%s&type=track&limit=1", apiBase, q))
		if err == nil && len(result.Tracks.Items) > 0 {
			return &result.Tracks.Items[0], nil
		}
	}

	// Fall back to title + artist search
	q := url.QueryEscape(fmt.Sprintf("track:%s artist:%s", track.Title, track.Artist))
	result, err := apiGet[spotifySearchResult](fmt.Sprintf("%s/search?q=%s&type=track&limit=1", apiBase, q))
	if err != nil {
		return nil, err
	}
	if len(result.Tracks.Items) == 0 {
		return nil, fmt.Errorf("track not found: %s — %s", track.Artist, track.Title)
	}
	return &result.Tracks.Items[0], nil
}

func getCurrentUser() (*spotifyUser, error) {
	return apiGet[spotifyUser](apiBase + "/me")
}

func createPlaylist(userID, name string) (*spotifyPlaylist, error) {
	return apiPost[spotifyPlaylist](
		fmt.Sprintf("%s/users/%s/playlists", apiBase, userID),
		createPlaylistRequest{Name: name, Public: false},
	)
}

func addTracksToPlaylist(playlistID string, uris []string) error {
	// Spotify allows max 100 tracks per request
	for i := 0; i < len(uris); i += 100 {
		end := min(i+100, len(uris))
		_, err := apiPost[json.RawMessage](
			fmt.Sprintf("%s/playlists/%s/tracks", apiBase, playlistID),
			addTracksRequest{URIs: uris[i:end]},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveToLibrary(ids []string) error {
	// Spotify allows max 50 IDs per request
	for i := 0; i < len(ids); i += 50 {
		end := min(i+50, len(ids))
		token, err := getAccessToken()
		if err != nil {
			return err
		}

		data, err := json.Marshal(map[string][]string{"ids": ids[i:end]})
		if err != nil {
			return err
		}

		req, err := http.NewRequest("PUT", apiBase+"/me/tracks", bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("spotify API error %d saving tracks", resp.StatusCode)
		}
	}
	return nil
}
