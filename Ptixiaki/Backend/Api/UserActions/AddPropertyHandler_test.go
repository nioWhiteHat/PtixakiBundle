package UserActions

import (
	p "Ptixiaki/Backend/Api"
	geo "Ptixiaki/Backend/ExternalServices/Geo"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const addPropertyTestInput = `{"SubType":"apartment","Address":"  Test street 12, Athens, Greece  ","SquareMeters":82,"ConstructionYear":2004,"FloorLevel":[3],"AddressOSMType":"way","AddressOSMID":123,"UID":"caller_uid","GeoLat":1,"GeoLon":2,"ZoneValue":3,"AdminName":"caller area"}`

func addPropertyRequest(method, transaction, body, userUID string) *http.Request {
	request := httptest.NewRequest(method, "/User/AddProperty?transaction="+transaction, strings.NewReader(body))
	if userUID != "" {
		request = request.WithContext(context.WithValue(request.Context(), "user_uid", userUID))
	}
	return request
}

func TestAddPropertyReturnsCreatedUIDAndSavedAddress(t *testing.T) {
	for _, transaction := range []string{"sell", "rent"} {
		t.Run(transaction, func(t *testing.T) {
			calls := 0
			create := func(_ context.Context, userUID string, prop p.PropertyFeatures, gotTransaction string) (string, error) {
				calls++
				if userUID != "owner" || gotTransaction != transaction {
					t.Fatal("creation must use the authenticated user and selected transaction")
				}
				if prop.AddressOSMType != "way" || prop.AddressOSMID != 123 || prop.Address != "Test street 12, Athens, Greece" || prop.SquareMeters != 82 || prop.AskingPricePerSqm != nil {
					t.Fatalf("lost saved inputs: %+v", prop)
				}
				if prop.UID != "" || prop.GeoLat != 0 || prop.GeoLon != 0 || prop.ZoneValue != 0 || prop.AdminName != "" {
					t.Fatal("caller identity and geography must be discarded")
				}
				return "property_created", nil
			}
			response := httptest.NewRecorder()
			ServeAddProperty(response, addPropertyRequest(http.MethodPost, transaction, addPropertyTestInput, "owner"), create)
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusCreated || calls != 1 || body["uid"] != "property_created" || body["message"] != "Property added successfully" {
				t.Fatalf("unexpected creation response: %d %s", response.Code, response.Body.String())
			}
			if response.Header().Get("Location") != "/User/ViewProperty?uid=property_created" {
				t.Fatal("Location must point to the created property's owner-only view")
			}
		})
	}
}

func TestAddPropertyRejectsInvalidRequestsBeforeCreation(t *testing.T) {
	cases := []struct {
		name, method, transaction, body, user string
		status                                int
	}{
		{"wrong method", http.MethodGet, "sell", addPropertyTestInput, "owner", 405},
		{"missing auth", http.MethodPost, "sell", addPropertyTestInput, "", 401},
		{"invalid transaction", http.MethodPost, "sale", addPropertyTestInput, "owner", 400},
		{"missing selection", http.MethodPost, "sell", strings.Replace(addPropertyTestInput, `,"AddressOSMType":"way","AddressOSMID":123`, "", 1), "owner", 422},
		{"invalid json", http.MethodPost, "sell", "broken", "owner", 400},
		{"missing address", http.MethodPost, "sell", strings.Replace(addPropertyTestInput, "  Test street 12, Athens, Greece  ", "   ", 1), "owner", 400},
		{"unknown field", http.MethodPost, "sell", strings.TrimSuffix(addPropertyTestInput, "}") + `,"extra":1}`, "owner", 400},
		{"second object", http.MethodPost, "sell", addPropertyTestInput + `{}`, "owner", 400},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			ServeAddProperty(response, addPropertyRequest(test.method, test.transaction, test.body, test.user), func(context.Context, string, p.PropertyFeatures, string) (string, error) {
				t.Fatal("invalid request reached creation")
				return "", nil
			})
			if response.Code != test.status {
				t.Fatalf("status %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestAddPropertyDoesNotExposeStorageErrorsOrRetryCreation(t *testing.T) {
	response := httptest.NewRecorder()
	calls := 0
	ServeAddProperty(response, addPropertyRequest(http.MethodPost, "sell", addPropertyTestInput, "owner"), func(context.Context, string, p.PropertyFeatures, string) (string, error) {
		calls++
		return "", errors.New("private database detail")
	})
	if response.Code != 500 || calls != 1 || strings.Contains(response.Body.String(), "private database detail") || !strings.Contains(response.Body.String(), "check your portfolio") {
		t.Fatalf("unsafe creation error: %d %s", response.Code, response.Body.String())
	}
}

func TestAddPropertyRejectsChangedAddressSelection(t *testing.T) {
	response := httptest.NewRecorder()
	ServeAddProperty(response, addPropertyRequest(http.MethodPost, "sell", addPropertyTestInput, "owner"), func(context.Context, string, p.PropertyFeatures, string) (string, error) {
		return "", geo.ErrAddressSelection
	})
	if response.Code != 422 || !strings.Contains(response.Body.String(), "address_selection_invalid") {
		t.Fatalf("unexpected address rejection: %d %s", response.Code, response.Body.String())
	}
}
