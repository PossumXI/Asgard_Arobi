// satellite_tracker demonstrates real-time satellite tracking integration.
// Uses free TLE API for orbit data and optional N2YO API for real-time positions.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/asgard/pandora/internal/platform/satellite"
)

func main() {
	// Command line flags
	noradID := flag.Int("norad", satellite.NoradISS, "NORAD catalog ID (default: ISS 25544)")
	n2yoKey := flag.String("n2yo-key", os.Getenv("N2YO_API_KEY"), "N2YO API key for real-time tracking")
	obsLat := flag.Float64("lat", 40.7128, "Observer latitude (default: NYC)")
	obsLon := flag.Float64("lon", -74.0060, "Observer longitude")
	obsAlt := flag.Float64("alt", 10, "Observer altitude in meters")
	duration := flag.Int("duration", 90, "Propagation duration in minutes")
	outputJSON := flag.Bool("json", false, "Output as JSON")
	showPasses := flag.Bool("passes", false, "Show upcoming visual passes (requires N2YO key)")
	flag.Parse()

	log.SetFlags(log.Ltime)
	ctx := context.Background()

	// Create satellite client
	cfg := satellite.DefaultConfig()
	cfg.N2YOAPIKey = *n2yoKey
	client := satellite.NewClient(cfg)

	// Fetch TLE
	log.Printf("Fetching TLE for NORAD ID %d...", *noradID)
	tle, err := client.GetTLE(ctx, *noradID)
	if err != nil {
		log.Fatalf("Failed to fetch TLE: %v", err)
	}

	log.Printf("Satellite: %s", tle.Name)
	log.Printf("TLE Line 1: %s", tle.Line1)
	log.Printf("TLE Line 2: %s", tle.Line2)
	log.Printf("Source: %s, Retrieved: %s", tle.Source, tle.RetrievedAt.Format(time.RFC3339))
	fmt.Println()

	// Create propagator
	propagator, err := satellite.NewPropagator(tle)
	if err != nil {
		log.Fatalf("Failed to create propagator: %v", err)
	}

	// Compute current position
	now := time.Now().UTC()
	lat, lon, alt := propagator.Propagate(now)
	log.Printf("Current Position (propagated):")
	log.Printf("  Latitude:  %.4f°", lat)
	log.Printf("  Longitude: %.4f°", lon)
	log.Printf("  Altitude:  %.2f km", alt)
	fmt.Println()

	// If N2YO key provided, fetch real-time position for comparison
	if *n2yoKey != "" {
		observer := satellite.Observer{
			Latitude:  *obsLat,
			Longitude: *obsLon,
			Altitude:  *obsAlt,
		}

		log.Printf("Fetching real-time position from N2YO API...")
		positions, err := client.GetPosition(ctx, *noradID, observer, 1)
		if err != nil {
			log.Printf("Warning: N2YO API error: %v", err)
		} else if len(positions) > 0 {
			p := positions[0]
			log.Printf("Real-time Position (N2YO):")
			log.Printf("  Latitude:  %.4f°", p.Latitude)
			log.Printf("  Longitude: %.4f°", p.Longitude)
			log.Printf("  Altitude:  %.2f km", p.Altitude)
			log.Printf("  Azimuth:   %.2f° (from observer)", p.Azimuth)
			log.Printf("  Elevation: %.2f° (from observer)", p.Elevation)
			log.Printf("  Eclipsed:  %v", p.Eclipsed)
			fmt.Println()
		}

		// Show visual passes if requested
		if *showPasses {
			log.Printf("Fetching upcoming visual passes...")
			passes, err := client.GetVisualPasses(ctx, *noradID, observer, 5, 60)
			if err != nil {
				log.Printf("Warning: Could not fetch passes: %v", err)
			} else if len(passes) > 0 {
				log.Printf("Upcoming Visual Passes:")
				for i, pass := range passes {
					log.Printf("  Pass %d:", i+1)
					log.Printf("    Start: %s (Az: %.1f°, El: %.1f°)",
						pass.StartTime.Format("Jan 02 15:04"), pass.StartAzimuth, pass.StartElevation)
					log.Printf("    Max:   %s (Az: %.1f°, El: %.1f°)",
						pass.MaxTime.Format("Jan 02 15:04"), pass.MaxAzimuth, pass.MaxElevation)
					log.Printf("    End:   %s (Az: %.1f°, El: %.1f°)",
						pass.EndTime.Format("Jan 02 15:04"), pass.EndAzimuth, pass.EndElevation)
					log.Printf("    Duration: %ds, Magnitude: %.1f", pass.Duration, pass.Magnitude)
				}
				fmt.Println()
			} else {
				log.Printf("No visual passes found in next 5 days")
			}
		}
	}

	// Propagate ground track
	log.Printf("Computing %d-minute ground track...", *duration)
	positions := propagator.PropagateRange(now, time.Duration(*duration)*time.Minute, time.Minute)

	if *outputJSON {
		output := struct {
			Satellite  string                         `json:"satellite"`
			NoradID    int                            `json:"norad_id"`
			TLE        *satellite.TLE                 `json:"tle"`
			Positions  []satellite.PropagatedPosition `json:"positions"`
			ComputedAt string                         `json:"computed_at"`
		}{
			Satellite:  tle.Name,
			NoradID:    *noradID,
			TLE:        tle,
			Positions:  positions,
			ComputedAt: now.Format(time.RFC3339),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	} else {
		log.Printf("Ground Track (every 10 minutes):")
		for i, pos := range positions {
			if i%10 == 0 {
				log.Printf("  T+%3dm: Lat %.2f°, Lon %.2f°, Alt %.0f km",
					i, pos.Latitude, pos.Longitude, pos.Altitude)
			}
		}
	}

	fmt.Println()
	log.Println("Satellite tracking complete.")

	// Print usage hints
	if *n2yoKey == "" {
		fmt.Println("\nTip: Set N2YO_API_KEY or use -n2yo-key for real-time tracking features.")
		fmt.Println("     Get a free API key at: https://www.n2yo.com/api/")
	}
}
