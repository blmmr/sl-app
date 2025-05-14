package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Site represents a transit site.
type Site struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Departure represents a single departure entry.
type Departure struct {
	Destination string `json:"destination"`
	Display     string `json:"display"`
	Line        struct {
		Designation string `json:"designation"`
	} `json:"line"`
}

// DeparturesResponse wraps the API response.
type DeparturesResponse struct {
	Departures []Departure `json:"departures"`
}

// fetchSiteID retrieves the site ID by name.
func fetchSiteID(name string) (int, error) {
	URL := "https://transport.integration.sl.se/v1/sites?expand=true"
	resp, err := http.Get(URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var sites []Site
	if err := json.NewDecoder(resp.Body).Decode(&sites); err != nil {
		return 0, err
	}

	// Normalize and search
	lowerName := strings.ToLower(name)
	for _, site := range sites {
		if strings.ToLower(site.Name) == lowerName {
			return site.ID, nil
		}
	}
	return 0, fmt.Errorf("site not found: %s", name)
}

// fetchDepartures retrieves departures for a given site ID.
func fetchDepartures(siteID int) ([]Departure, error) {
	URL := fmt.Sprintf("https://transport.integration.sl.se/v1/sites/%d/departures", siteID)
	resp, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var departuresResp DeparturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&departuresResp); err != nil {
		return nil, err
	}

	return departuresResp.Departures, nil
}

// readSiteNames reads site names from sites.json.
func readSiteNames(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data struct {
		Sites []string `json:"sites"`
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	return data.Sites, nil
}

func main() {
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLFiles("templates/index.html")

	r.GET("/", func(c *gin.Context) {
		sites, err := readSiteNames("sites.json")
		if err != nil {
			log.Fatalf("Error reading sites: %v", err)
		}

		var results []struct {
			Site       string
			Departures []Departure
		}

		for _, siteName := range sites {
			siteID, err := fetchSiteID(siteName)
			if err != nil {
				log.Printf("Error fetching site ID for %s: %v\n", siteName, err)
				continue
			}

			departures, err := fetchDepartures(siteID)
			if err != nil {
				log.Printf("Error fetching departures for %s: %v\n", siteName, err)
				continue
			}

			results = append(results, struct {
				Site       string
				Departures []Departure
			}{
				Site:       siteName,
				Departures: departures,
			})
		}

		// Render HTML template with data
		c.HTML(http.StatusOK, "index.html", gin.H{
			"results": results,
		})
	})

	r.Run(":8080")
}
