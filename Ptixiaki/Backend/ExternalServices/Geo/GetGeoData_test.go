package GetGeoData

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testPlace = Place{
	Lat: "37.9679634", Lon: "23.7248963", DisplayName: "Ερεχθείου, Αθήνα, Ελλάδα",
	OSMType: "way", OSMID: 123, Address: map[string]string{"road": "Ερεχθείου", "city": "Αθήνα"},
}

func TestNominatimSearchAndSaveUseTheSameLocation(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.URL.Query()
		if r.URL.Path != "/search" || query.Get("q") != "ερεχθειου" || query.Get("limit") != "8" || query.Get("format") != "json" || query.Get("addressdetails") != "1" {
			t.Errorf("unexpected Nominatim request: %s", r.URL)
		}
		if !strings.Contains(r.UserAgent(), "PropertyCoPilot") {
			t.Error("Nominatim needs the application User-Agent")
		}
		json.NewEncoder(w).Encode([]Place{testPlace})
	}))
	defer server.Close()
	client := &NominatimClient{BaseURL: server.URL, Client: server.Client()}
	places, err := client.Search(context.Background(), " ερεχθειου ")
	if err != nil || len(places) != 1 || places[0].OSMID != testPlace.OSMID || places[0].Address["road"] != "Ερεχθείου" {
		t.Fatalf("search did not return the provider location: %+v, %v", places, err)
	}
	saved, err := client.Resolve(context.Background(), "way", 123, testPlace.DisplayName)
	if err != nil || saved.Lat != places[0].Lat || saved.Lon != places[0].Lon {
		t.Fatalf("saving changed the selected coordinates: %+v, %v", saved, err)
	}
	if _, err := client.Resolve(context.Background(), "way", 123, "Another address"); !errors.Is(err, ErrAddressSelection) {
		t.Fatalf("changed address accepted: %v", err)
	}
	if _, err := client.Search(context.Background(), "ερεχθειου"); err != nil || calls != 1 {
		t.Fatalf("cached search/selection made extra requests: %d, %v", calls, err)
	}
}

func TestNominatimResolvesSelectionAfterCacheExpires(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lookup" || r.URL.Query().Get("osm_ids") != "W123" {
			t.Errorf("unexpected lookup: %s", r.URL)
		}
		json.NewEncoder(w).Encode([]Place{testPlace})
	}))
	defer server.Close()
	client := &NominatimClient{BaseURL: server.URL, Client: server.Client()}
	place, err := client.Resolve(context.Background(), "way", 123, testPlace.DisplayName)
	if err != nil || place.Lat != testPlace.Lat || place.Lon != testPlace.Lon {
		t.Fatalf("lookup failed: %+v, %v", place, err)
	}
	for _, osmType := range []string{"", "invalid"} {
		if _, err := client.Resolve(context.Background(), osmType, 123, testPlace.DisplayName); !errors.Is(err, ErrAddressSelection) {
			t.Fatalf("invalid OSM type accepted: %v", err)
		}
	}
}

func TestNominatimRejectsInvalidProviderCoordinates(t *testing.T) {
	invalid := []Place{testPlace, testPlace, testPlace, testPlace}
	invalid[0].Lat = "NaN"
	invalid[1].Lon = "181"
	invalid[2].OSMID = 0
	invalid[3].OSMType = "invalid"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(invalid)
	}))
	defer server.Close()
	client := &NominatimClient{BaseURL: server.URL, Client: server.Client()}
	places, err := client.Search(context.Background(), "ερεχθειου")
	if err != nil || len(places) != 0 {
		t.Fatalf("invalid provider locations accepted: %+v, %v", places, err)
	}
}

func TestNominatimProviderFailuresAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := &NominatimClient{BaseURL: server.URL, Client: server.Client()}
	if _, err := client.Search(context.Background(), "ερεχθειου"); err == nil {
		t.Fatal("provider failure must not become an empty successful search")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Search(ctx, "Αθήνα"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled search should stop while waiting for provider rate limit: %v", err)
	}
}

func TestNominatimRateLimitsDifferentSearches(t *testing.T) {
	var requests []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, time.Now())
		json.NewEncoder(w).Encode([]Place{testPlace})
	}))
	defer server.Close()
	client := &NominatimClient{BaseURL: server.URL, Client: server.Client()}
	for _, address := range []string{"Ερεχθείου Αθήνα", "Ερεχθείου Άλιμος"} {
		if _, err := client.Search(context.Background(), address); err != nil {
			t.Fatal(err)
		}
	}
	if len(requests) != 2 || requests[1].Sub(requests[0]) < 990*time.Millisecond {
		t.Fatalf("provider requests must be at least one second apart: %v", requests)
	}
}
