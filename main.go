package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"nmft-plugin/sonicsimilarity"
)

const configAPIUrl = "apiUrl"
const defaultAPIUrl = "http://localhost:8100"
const configSimilarityThreshold = "similarityThreshold"
const defaultSimilarityThreshold = 0.98

// Compile-time check that we implement necessary interfaces
var _ metadata.SimilarSongsByTrackProvider = (*nmftPlugin)(nil)
var _ metadata.SimilarArtistsProvider = (*nmftPlugin)(nil)
var _ sonicsimilarity.SonicSimilarity = (*nmftPlugin)(nil)

// ---- API Response Structs (from OpenAPI spec) ----

type trackNeighbor struct {
	SubsonicID string  `json:"subsonic_id"`
	Title      string  `json:"title"`
	AlbumName  string  `json:"album_name"`
	ArtistID   string  `json:"artist_id"`
	ArtistName string  `json:"artist_name"`
	Distance   float64 `json:"distance"`
}

type artistNeighbor struct {
	ArtistID string  `json:"artist_id"`
	Distance float64 `json:"distance"`
}

type trackOut struct {
	SubsonicID string `json:"subsonic_id"`
	Title      string `json:"title"`
	AlbumName  string `json:"album_name"`
	ArtistID   string `json:"artist_id"`
	ArtistName string `json:"artist_name"`
}

type nmftPlugin struct{}

func init() {
	metadata.Register(&nmftPlugin{})
	sonicsimilarity.Register(&nmftPlugin{})
	pdk.Log(pdk.LogInfo, "[Absolute Pitch] Plugin registered successfully")
}

func getConfigString(key, defaultValue string) string {
	if value, ok := pdk.GetConfig(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func getConfigFloat(key string, defaultValue float64) float64 {
	if value, ok := pdk.GetConfig(key); ok && value != "" {
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			return val
		}
	}
	return defaultValue
}

func apiBase() string {
	return getConfigString(configAPIUrl, defaultAPIUrl)
}

func httpGetJSON(url string, dst any) error {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("[Absolute Pitch] GET %s", url))
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method: "GET",
		URL:    url,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		pdk.Log(pdk.LogError, fmt.Sprintf("[Absolute Pitch] HTTP request failed: %v", err))
		return err
	}
	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("[Absolute Pitch] API returned status %d: %s", resp.StatusCode, resp.Body)
		pdk.Log(pdk.LogError, errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	return json.Unmarshal(resp.Body, dst)
}

// ---- MetadataAgent Implementations ----

// GetSimilarSongsByTrack -> Subsonic getSimilarSongs
func (p *nmftPlugin) GetSimilarSongsByTrack(input metadata.SimilarSongsByTrackRequest) (*metadata.SimilarSongsResponse, error) {
	count := int(input.Count)
	if count <= 0 {
		count = 20
	}
	threshold := getConfigFloat(configSimilarityThreshold, defaultSimilarityThreshold)
	limit := count + 10

	url := fmt.Sprintf("%s/tracks/%s/neighbors?limit=%d&deduplicate=true", apiBase(), url.PathEscape(input.ID), limit)

	var apiResp []trackNeighbor
	if err := httpGetJSON(url, &apiResp); err != nil {
		return &metadata.SimilarSongsResponse{Songs: []metadata.SongRef{}}, nil // Fail gracefully
	}

	songs := make([]metadata.SongRef, 0, len(apiResp))
	for _, t := range apiResp {
		similarity := normalizeSimilarity(t.Distance)
		if similarity >= threshold {
			continue
		}
		songs = append(songs, metadata.SongRef{
			ID:     t.SubsonicID,
			Name:   t.Title,
			Artist: t.ArtistName,
			Album:  t.AlbumName,
		})
		if len(songs) >= count {
			break
		}
	}
	return &metadata.SimilarSongsResponse{Songs: songs}, nil
}

// GetSimilarArtists -> Subsonic getSimilarArtists
func (p *nmftPlugin) GetSimilarArtists(input metadata.SimilarArtistsRequest) (*metadata.SimilarArtistsResponse, error) {
	limit := int(input.Limit)
	if limit <= 0 {
		limit = 10
	}
	url := fmt.Sprintf("%s/artists/%s/neighbors?limit=%d", apiBase(), url.PathEscape(input.ID), limit)

	var apiResp []artistNeighbor
	if err := httpGetJSON(url, &apiResp); err != nil {
		return &metadata.SimilarArtistsResponse{Artists: []metadata.ArtistRef{}}, nil
	}

	artists := make([]metadata.ArtistRef, 0, len(apiResp))
	for _, a := range apiResp {
		artists = append(artists, metadata.ArtistRef{
			ID:   a.ArtistID,
			Name: "", // Left empty; host resolves name and other metadata from DB using ID
		})
	}
	return &metadata.SimilarArtistsResponse{Artists: artists}, nil
}

// ---- SonicSimilarity Implementations ----

func normalizeSimilarity(distance float64) float64 {
	similarity := 1.0 - distance
	if similarity < 0 {
		similarity = 0
	}
	if similarity > 1 {
		similarity = 1
	}
	return similarity
}

func (p *nmftPlugin) GetSonicSimilarTracks(input sonicsimilarity.GetSonicSimilarTracksRequest) (sonicsimilarity.SonicSimilarityResponse, error) {
	if input.Song.ID == "" {
		return sonicsimilarity.SonicSimilarityResponse{}, fmt.Errorf("song.id is required")
	}

	count := int(input.Count)
	if count <= 0 {
		count = 10
	}
	threshold := getConfigFloat(configSimilarityThreshold, defaultSimilarityThreshold)
	limit := count + 10

	url := fmt.Sprintf("%s/tracks/%s/neighbors?limit=%d", apiBase(), url.PathEscape(input.Song.ID), limit)

	var apiResp []trackNeighbor
	if err := httpGetJSON(url, &apiResp); err != nil {
		return sonicsimilarity.SonicSimilarityResponse{}, err
	}

	matches := make([]sonicsimilarity.SonicMatch, 0, len(apiResp))
	for _, t := range apiResp {
		similarity := normalizeSimilarity(t.Distance)
		if similarity >= threshold {
			continue
		}
		matches = append(matches, sonicsimilarity.SonicMatch{
			Song: metadata.SongRef{
				ID:     t.SubsonicID,
				Name:   t.Title,
				Artist: t.ArtistName,
				Album:  t.AlbumName,
			},
			Similarity: similarity,
		})
		if len(matches) >= count {
			break
		}
	}

	return sonicsimilarity.SonicSimilarityResponse{Matches: matches}, nil
}

func (p *nmftPlugin) FindSonicPath(input sonicsimilarity.FindSonicPathRequest) (sonicsimilarity.SonicSimilarityResponse, error) {
	if input.StartSong.ID == "" || input.EndSong.ID == "" {
		return sonicsimilarity.SonicSimilarityResponse{}, fmt.Errorf("startSong.id and endSong.id are required")
	}

	url := fmt.Sprintf("%s/sonic-path?start=%s&end=%s", apiBase(), url.QueryEscape(input.StartSong.ID), url.QueryEscape(input.EndSong.ID))

	var apiResp []trackOut
	if err := httpGetJSON(url, &apiResp); err != nil {
		return sonicsimilarity.SonicSimilarityResponse{}, err
	}

	matches := make([]sonicsimilarity.SonicMatch, 0, len(apiResp))
	for i, t := range apiResp {
		// Assign a progressive similarity score from 1.0 down to 0.5 to preserve ordering
		similarity := 1.0
		if len(apiResp) > 1 {
			similarity = 1.0 - (float64(i)/float64(len(apiResp)-1))*0.5
		}
		matches = append(matches, sonicsimilarity.SonicMatch{
			Song: metadata.SongRef{
				ID:     t.SubsonicID,
				Name:   t.Title,
				Artist: t.ArtistName,
				Album:  t.AlbumName,
			},
			Similarity: similarity,
		})
	}

	return sonicsimilarity.SonicSimilarityResponse{Matches: matches}, nil
}

func main() {}
