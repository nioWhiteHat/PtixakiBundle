package main

import (
	auth "Ptixiaki/Backend/Auth"
	geo "Ptixiaki/Backend/ExternalServices/Geo"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalysisHandlerRejectsBadAreaBeforeDatabase(t *testing.T) {
	app := application{}
	for _, body := range []string{"broken", `{}`, `{"MinLat":38,"MaxLat":37,"MinLon":23,"MaxLon":24}`} {
		w := httptest.NewRecorder()
		app.CreateAnalysisForAreaHandler(w, httptest.NewRequest(http.MethodPost, "/analysis/CreateAnalysisForArea", strings.NewReader(body)))
		var response struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 400 || response.Error.Code != "invalid_area" {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}

func TestAnalysisRoutesStillRequireAuthentication(t *testing.T) {
	app := application{}
	for _, handler := range []http.HandlerFunc{app.AutocompleteHandler, app.CreateAnalysisForAreaHandler} {
		w := httptest.NewRecorder()
		auth.Authenticate(handler)(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d", w.Code)
		}
	}
}

func TestAddressSearchUsesNominatimWithoutSavedProperties(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" || r.URL.Query().Get("q") != "ερεχθειου" {
			t.Errorf("wrong provider query: %s", r.URL)
		}
		json.NewEncoder(w).Encode([]geo.Place{{
			DisplayName: "Ερεχθείου, Αθήνα, Ελλάδα", Lat: "37.9679634", Lon: "23.7248963", OSMID: 123, OSMType: "way",
		}})
	}))
	defer server.Close()
	previous := geo.Geocoder
	geo.Geocoder = &geo.NominatimClient{BaseURL: server.URL, Client: server.Client()}
	defer func() { geo.Geocoder = previous }()
	app := application{}
	response := httptest.NewRecorder()
	app.AutocompleteHandler(response, httptest.NewRequest(http.MethodGet, "/analysis/autocomplete?q=%CE%B5%CF%81%CE%B5%CF%87%CE%B8%CE%B5%CE%B9%CE%BF%CF%85&kind=address", nil))
	var places []geo.Place
	if err := json.Unmarshal(response.Body.Bytes(), &places); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(places) != 1 || places[0].OSMID != 123 || places[0].Lat != "37.9679634" {
		t.Fatalf("address search failed without a database: %d %s", response.Code, response.Body.String())
	}
}
