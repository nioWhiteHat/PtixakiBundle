package GetGeoData

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Place struct {
	Lat         string            `json:"lat"`
	Lon         string            `json:"lon"`
	DisplayName string            `json:"display_name"`
	Type        string            `json:"type"`
	OSMType     string            `json:"osm_type"`
	OSMID       int64             `json:"osm_id"`
	BoundingBox []string          `json:"boundingbox"`
	Address     map[string]string `json:"address"`
}

var ErrAddressSelection = errors.New("selected Nominatim address could not be verified")
var Geocoder = &NominatimClient{}

type geocodingCache struct {
	places  []Place
	expires time.Time
}

type NominatimClient struct {
	BaseURL     string
	Client      *http.Client
	mu          sync.Mutex
	lastRequest time.Time
	cache       map[string]geocodingCache
}

func validPlace(place Place) bool {
	lat, latErr := strconv.ParseFloat(place.Lat, 64)
	lon, lonErr := strconv.ParseFloat(place.Lon, 64)
	return latErr == nil && lonErr == nil && !math.IsNaN(lat) && !math.IsInf(lat, 0) && math.Abs(lat) <= 90 &&
		!math.IsNaN(lon) && !math.IsInf(lon, 0) && math.Abs(lon) <= 180 &&
		strings.TrimSpace(place.DisplayName) != "" && utf8.RuneCountInString(place.DisplayName) <= 255 &&
		place.OSMID > 0 && (place.OSMType == "node" || place.OSMType == "way" || place.OSMType == "relation")
}

func (client *NominatimClient) Search(ctx context.Context, address string) ([]Place, error) {
	address = strings.TrimSpace(address)
	if utf8.RuneCountInString(address) < 2 || utf8.RuneCountInString(address) > 255 {
		return nil, fmt.Errorf("address must contain between 2 and 255 characters")
	}
	return client.request(ctx, "search", url.Values{"q": {address}, "limit": {"8"}})
}

func (client *NominatimClient) Resolve(ctx context.Context, osmType string, osmID int64, address string) (Place, error) {
	prefixes := map[string]string{"node": "N", "way": "W", "relation": "R"}
	prefix := prefixes[osmType]
	if prefix == "" || osmID <= 0 {
		return Place{}, ErrAddressSelection
	}
	places, err := client.request(ctx, "lookup", url.Values{"osm_ids": {prefix + strconv.FormatInt(osmID, 10)}})
	if err != nil {
		return Place{}, err
	}
	for _, place := range places {
		if place.OSMType == osmType && place.OSMID == osmID && place.DisplayName == strings.TrimSpace(address) {
			return place, nil
		}
	}
	return Place{}, ErrAddressSelection
}

func (client *NominatimClient) request(ctx context.Context, path string, parameters url.Values) ([]Place, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	key := path + "?" + parameters.Encode()
	if entry, ok := client.cache[key]; ok && time.Now().Before(entry.expires) {
		return entry.places, nil
	}
	timer := time.NewTimer(max(0, time.Until(client.lastRequest.Add(time.Second))))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	baseURL := client.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("NOMINATIM_URL")
	}
	if baseURL == "" {
		baseURL = "https://nominatim.openstreetmap.org"
	}
	parameters.Set("format", "json")
	parameters.Set("addressdetails", "1")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/"+path+"?"+parameters.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "PropertyCoPilot/1.0 (user-submitted address search)")
	request.Header.Set("Accept", "application/json")
	httpClient := client.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 12 * time.Second}
	}
	client.lastRequest = time.Now()
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim returned HTTP %d", response.StatusCode)
	}
	var places []Place
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&places); err != nil {
		return nil, err
	}
	results := make([]Place, 0, len(places))
	for _, place := range places {
		if validPlace(place) {
			results = append(results, place)
		}
	}
	client.remember(key, results)
	for _, place := range results {
		prefix := map[string]string{"node": "N", "way": "W", "relation": "R"}[place.OSMType]
		lookup := "lookup?" + url.Values{"osm_ids": {prefix + strconv.FormatInt(place.OSMID, 10)}}.Encode()
		client.remember(lookup, []Place{place})
	}
	return results, nil
}

func (client *NominatimClient) remember(key string, places []Place) {
	if client.cache == nil {
		client.cache = make(map[string]geocodingCache)
	}
	if len(client.cache) >= 256 {
		for oldKey := range client.cache {
			delete(client.cache, oldKey)
			break
		}
	}
	client.cache[key] = geocodingCache{places, time.Now().Add(time.Hour)}
}

func GetGeoData(address string) (Place, error) {
	places, err := Geocoder.Search(context.Background(), address)
	if err != nil {
		return Place{}, err
	}
	if len(places) == 0 {
		return Place{}, ErrAddressSelection
	}
	return places[0], nil
}

func (g *Geo) GetZoneValue(lat, lon string) (int, error) {
	latF, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return 0, err
	}
	lonF, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return 0, err
	}
	query := `SELECT b.zone_value FROM property_geo_data g
 JOIN base_attributes b ON g.property_uid = b.property_uid
 ORDER BY g.geom <-> ST_SetSRID(ST_MakePoint($1, $2), 4326) LIMIT 1`
	var zoneValue int
	err = g.Db.QueryRow(g.Ctx, query, lonF, latF).Scan(&zoneValue)
	return zoneValue, err
}
