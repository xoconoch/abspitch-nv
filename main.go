package main

import (
    "encoding/json"
    "fmt"

    "github.com/navidrome/navidrome/plugins/pdk/go/host"
    "github.com/navidrome/navidrome/plugins/pdk/go/metadata"
    "github.com/navidrome/navidrome/plugins/pdk/go/pdk"
    "nmft-plugin/sonicsimilarity"
)

const configAPIUrl = "apiUrl"
const defaultAPIUrl = "http://localhost:8000"

// Compile-time check that we implement necessary interfaces
var _ metadata.SimilarSongsByArtistProvider = (*nmftPlugin)(nil)
var _ metadata.SimilarSongsByTrackProvider = (*nmftPlugin)(nil)
var _ metadata.SimilarArtistsProvider = (*nmftPlugin)(nil)
var _ sonicsimilarity.SonicSimilarity = (*nmftPlugin)(nil)

// ---- API Response Structs (from OpenAPI spec) ----

type similarTrack struct {
    SubsonicID string  `json:"subsonic_id"`
    Title      string  `json:"title"`
    Artist     string  `json:"artist"`
    Album      *string `json:"album"`
    Duration   *int    `json:"duration"`
    Path       *string `json:"path"`
    Distance   float64 `json:"distance"`
}

type similarResponse struct {
    QueryTrackID string         `json:"query_track_id"`
    QueryTrack   *similarTrack  `json:"query_track"`
    Results      []similarTrack `json:"results"`
    Count        int            `json:"count"`
}

type similarArtist struct {
    SubsonicID string  `json:"subsonic_id"`
    Name       string  `json:"name"`
    Distance   float64 `json:"distance"`
}

type similarArtistsResponse struct {
    QueryArtistID string          `json:"query_artist_id"`
    Results       []similarArtist `json:"results"`
    Count         int             `json:"count"`
}

type nmftPlugin struct{}

func init() {
    metadata.Register(&nmftPlugin{})
    sonicsimilarity.Register(&nmftPlugin{})
    pdk.Log(pdk.LogInfo, "[NMFT] Plugin registered successfully")
}

func getConfigString(key, defaultValue string) string {
    if value, ok := pdk.GetConfig(key); ok && value != "" {
        return value
    }
    return defaultValue
}

func apiBase() string {
    return getConfigString(configAPIUrl, defaultAPIUrl)
}

func httpGetJSON(url string, dst any) error {
    pdk.Log(pdk.LogInfo, fmt.Sprintf("[NMFT] GET %s", url))
    resp, err := host.HTTPSend(host.HTTPRequest{
        Method: "GET",
        URL:    url,
        Headers: map[string]string{
            "Accept": "application/json",
        },
    })
    if err != nil {
        pdk.Log(pdk.LogError, fmt.Sprintf("[NMFT] HTTP request failed: %v", err))
        return err
    }
    if resp.StatusCode != 200 {
        errMsg := fmt.Sprintf("[NMFT] API returned status %d: %s", resp.StatusCode, resp.Body)
        pdk.Log(pdk.LogError, errMsg)
        return fmt.Errorf(errMsg)
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
    url := fmt.Sprintf("%s/similar/%s?limit=%d", apiBase(), input.ID, count)

    var apiResp similarResponse
    if err := httpGetJSON(url, &apiResp); err != nil {
        return &metadata.SimilarSongsResponse{Songs: []metadata.SongRef{}}, nil // Fail gracefully
    }

    songs := make([]metadata.SongRef, 0, len(apiResp.Results))
    for _, t := range apiResp.Results {
        album := ""
        if t.Album != nil {
            album = *t.Album
        }
        songs = append(songs, metadata.SongRef{
            ID:     t.SubsonicID,
            Name:   t.Title,
            Artist: t.Artist,
            Album:  album,
        })
    }
    return &metadata.SimilarSongsResponse{Songs: songs}, nil
}

// GetSimilarSongsByArtist -> Subsonic getSimilarSongs2
func (p *nmftPlugin) GetSimilarSongsByArtist(input metadata.SimilarSongsByArtistRequest) (*metadata.SimilarSongsResponse, error) {
    count := int(input.Count)
    if count <= 0 {
        count = 20
    }
    url := fmt.Sprintf("%s/similar_tracks_by_artist/%s?limit=%d", apiBase(), input.ID, count)

    var apiResp similarResponse
    if err := httpGetJSON(url, &apiResp); err != nil {
        return &metadata.SimilarSongsResponse{Songs: []metadata.SongRef{}}, nil
    }

    songs := make([]metadata.SongRef, 0, len(apiResp.Results))
    for _, t := range apiResp.Results {
        album := ""
        if t.Album != nil {
            album = *t.Album
        }
        songs = append(songs, metadata.SongRef{
            ID:     t.SubsonicID,
            Name:   t.Title,
            Artist: t.Artist,
            Album:  album,
        })
    }
    return &metadata.SimilarSongsResponse{Songs: songs}, nil
}

// GetSimilarArtists -> Subsonic getSimilarArtists
func (p *nmftPlugin) GetSimilarArtists(input metadata.SimilarArtistsRequest) (*metadata.SimilarArtistsResponse, error) {
    limit := int(input.Limit)
    if limit <= 0 {
        limit = 10
    }
    url := fmt.Sprintf("%s/similar_artists/%s?limit=%d", apiBase(), input.ID, limit)

    var apiResp similarArtistsResponse
    if err := httpGetJSON(url, &apiResp); err != nil {
        return &metadata.SimilarArtistsResponse{Artists: []metadata.ArtistRef{}}, nil
    }

    artists := make([]metadata.ArtistRef, 0, len(apiResp.Results))
    for _, a := range apiResp.Results {
        artists = append(artists, metadata.ArtistRef{
            ID:   a.SubsonicID,
            Name: a.Name,
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

    url := fmt.Sprintf("%s/similar/%s?limit=%d", apiBase(), input.Song.ID, count)

    var apiResp similarResponse
    if err := httpGetJSON(url, &apiResp); err != nil {
        return sonicsimilarity.SonicSimilarityResponse{}, err
    }

    matches := make([]sonicsimilarity.SonicMatch, 0, len(apiResp.Results))
    for _, t := range apiResp.Results {
        album := ""
        if t.Album != nil {
            album = *t.Album
        }
        matches = append(matches, sonicsimilarity.SonicMatch{
            Song: metadata.SongRef{
                ID:     t.SubsonicID,
                Name:   t.Title,
                Artist: t.Artist,
                Album:  album,
            },
            Similarity: normalizeSimilarity(t.Distance),
        })
    }

    return sonicsimilarity.SonicSimilarityResponse{Matches: matches}, nil
}

func (p *nmftPlugin) FindSonicPath(input sonicsimilarity.FindSonicPathRequest) (sonicsimilarity.SonicSimilarityResponse, error) {
    // The NMFT OpenAPI spec does not contain a path-finding endpoint.
    // We return an error to indicate this feature is not supported by the backend.
    return sonicsimilarity.SonicSimilarityResponse{}, fmt.Errorf("find_sonic_path is not supported by Not My Fucking Tempo API")
}

func main() {}
