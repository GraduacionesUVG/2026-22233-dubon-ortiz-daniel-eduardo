package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// youtubeSearchResponse models the relevant parts of the YouTube Data API v3
// search response
type youtubeSearchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

// SearchExerciseVideo searches YouTube for a tutorial video matching the given exercise query
func SearchExerciseVideo(query string) string {
	searchTerm := query
	fallbackURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", url.QueryEscape(searchTerm))

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		return fallbackURL
	}

	requestURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&type=video&key=%s&maxResults=1",
		url.QueryEscape(searchTerm),
		apiKey,
	)

	httpClient := &http.Client{Timeout: 10 * time.Second}

	resp, err := httpClient.Get(requestURL)
	if err != nil {
		return fallbackURL
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackURL
	}

	var searchResp youtubeSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return fallbackURL
	}

	if len(searchResp.Items) == 0 || searchResp.Items[0].ID.VideoID == "" {
		return fallbackURL
	}

	return "https://www.youtube.com/watch?v=" + searchResp.Items[0].ID.VideoID
}
