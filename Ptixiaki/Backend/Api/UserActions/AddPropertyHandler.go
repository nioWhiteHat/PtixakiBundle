package UserActions

import (
	p "Ptixiaki/Backend/Api"
	geo "Ptixiaki/Backend/ExternalServices/Geo"
	explain "Ptixiaki/Worker/explainability"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

type PropertyCreator func(context.Context, string, p.PropertyFeatures, string) (string, error)

func ServeAddProperty(w http.ResponseWriter, r *http.Request, create PropertyCreator) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userUID, ok := r.Context().Value("user_uid").(string)
	if !ok || strings.TrimSpace(userUID) == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	transaction := r.URL.Query().Get("transaction")
	if err := explain.ValidateTransaction(transaction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prop, err := p.DecodePropertyInput(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if prop.AddressOSMID <= 0 || (prop.AddressOSMType != "node" && prop.AddressOSMType != "way" && prop.AddressOSMType != "relation") {
		propertyAddressError(w, "address_selection_required")
		return
	}
	prop.Address = strings.TrimSpace(prop.Address)
	prop.UID = ""
	prop.ZoneValue = 0
	prop.GeoLat = 0
	prop.GeoLon = 0
	prop.Distance = 0
	prop.AdminName = ""
	uid, err := create(r.Context(), userUID, prop, transaction)
	if errors.Is(err, geo.ErrAddressSelection) {
		propertyAddressError(w, "address_selection_invalid")
		return
	}
	if err != nil {
		log.Printf("failed to add property: %v", err)
		http.Error(w, "could not finish adding property; check your portfolio before trying again", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/User/ViewProperty?uid="+uid)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Property added successfully", "uid": uid}); err != nil {
		log.Printf("failed to encode add-property response: %v", err)
	}
}

func propertyAddressError(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code}})
}
