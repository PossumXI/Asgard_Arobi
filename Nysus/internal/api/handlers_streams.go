package api

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StreamResponse represents a video stream.
type StreamResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Source      string  `json:"source"`
	SourceType  string  `json:"sourceType"`
	SourceID    string  `json:"sourceId"`
	Location    string  `json:"location"`
	GeoLocation *GeoLoc `json:"geoLocation,omitempty"`
	Type        string  `json:"type"` // civilian, military, interstellar
	Status      string  `json:"status"` // live, delayed, offline
	Viewers     int     `json:"viewers"`
	Latency     int     `json:"latency"` // ms
	Thumbnail   string  `json:"thumbnail,omitempty"`
	Description string  `json:"description,omitempty"`
	Resolution  string  `json:"resolution"`
	Bitrate     int     `json:"bitrate"`
	StartedAt   string  `json:"startedAt"`
}

// StreamStats represents streaming statistics.
type StreamStats struct {
	TotalStreams int            `json:"totalStreams"`
	LiveStreams  int            `json:"liveStreams"`
	TotalViewers int            `json:"totalViewers"`
	ByCategory   map[string]int `json:"byCategory"`
}

// generateStreams creates sample stream data.
func generateStreams(count int, streamType string) []StreamResponse {
	streams := []StreamResponse{}

	sources := map[string][]struct {
		name     string
		location string
		lat, lon float64
	}{
		"civilian": {
			{"Pacific Wildfire Monitor", "California, USA", 36.7783, -119.4179},
			{"Atlantic Hurricane Watch", "Florida, USA", 27.9944, -81.7603},
			{"European Flood Alert", "Netherlands", 52.1326, 5.2913},
			{"Asian Typhoon Tracker", "Philippines", 12.8797, 121.7740},
			{"African Drought Monitor", "Kenya", -0.0236, 37.9062},
		},
		"military": {
			{"Strategic Surveillance Alpha", "Classified", 0, 0},
			{"Border Security Delta", "Classified", 0, 0},
			{"Naval Operations Echo", "Pacific Ocean", 20.0, -140.0},
		},
		"interstellar": {
			{"Mars Perseverance Relay", "Jezero Crater, Mars", 0, 0},
			{"Lunar Gateway Station", "Moon Orbit", 0, 0},
			{"Deep Space Network Feed", "Goldstone, CA", 35.4267, -116.8900},
		},
	}

	types := []string{streamType}
	if streamType == "" {
		types = []string{"civilian", "military", "interstellar"}
	}

	for _, t := range types {
		srcList := sources[t]
		for i := 0; i < count/len(types) && i < len(srcList); i++ {
			src := srcList[i]
			
			var geo *GeoLoc
			if src.lat != 0 || src.lon != 0 {
				geo = &GeoLoc{Latitude: src.lat, Longitude: src.lon}
			}

			latency := 50 + rand.Intn(200)
			if t == "interstellar" {
				latency = 180000 + rand.Intn(600000) // Mars: 3-13 min delay
			}

			status := "live"
			if rand.Float64() < 0.2 {
				status = []string{"delayed", "offline"}[rand.Intn(2)]
			}

			streams = append(streams, StreamResponse{
				ID:          uuid.New().String(),
				Title:       src.name,
				Source:      "ASGARD Satellite Network",
				SourceType:  "satellite",
				SourceID:    "sat-" + uuid.New().String()[:8],
				Location:    src.location,
				GeoLocation: geo,
				Type:        t,
				Status:      status,
				Viewers:     rand.Intn(10000) + 100,
				Latency:     latency,
				Description: "Real-time monitoring feed from ASGARD satellite constellation",
				Resolution:  []string{"1080p", "4K", "720p"}[rand.Intn(3)],
				Bitrate:     2000000 + rand.Intn(8000000),
				StartedAt:   time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second).Format(time.RFC3339),
			})
		}
	}

	return streams
}

// handleStreams handles GET /api/streams
func (s *Server) handleStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	streamType := r.URL.Query().Get("type")
	streams := generateStreams(15, streamType)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"streams": streams,
		"total":   len(streams),
	})
}

// handleStreamStats handles GET /api/streams/stats
func (s *Server) handleStreamStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	s.writeJSON(w, http.StatusOK, StreamStats{
		TotalStreams: 47,
		LiveStreams:  42,
		TotalViewers: 15000 + rand.Intn(5000),
		ByCategory: map[string]int{
			"civilian":     28,
			"military":     12,
			"interstellar": 7,
		},
	})
}

// handleFeaturedStreams handles GET /api/streams/featured
func (s *Server) handleFeaturedStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	streams := generateStreams(6, "")
	// Mark as featured (higher viewers)
	for i := range streams {
		streams[i].Viewers = 5000 + rand.Intn(20000)
		streams[i].Status = "live"
	}

	s.writeJSON(w, http.StatusOK, streams)
}

// handleStreamSearch handles GET /api/streams/search
func (s *Server) handleStreamSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	query := strings.ToLower(r.URL.Query().Get("q"))
	if query == "" {
		s.writeJSON(w, http.StatusOK, []StreamResponse{})
		return
	}

	streams := generateStreams(15, "")
	
	// Filter by query
	var results []StreamResponse
	for _, stream := range streams {
		if strings.Contains(strings.ToLower(stream.Title), query) ||
			strings.Contains(strings.ToLower(stream.Location), query) {
			results = append(results, stream)
		}
	}

	s.writeJSON(w, http.StatusOK, results)
}
