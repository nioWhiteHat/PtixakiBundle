package UserActions

import (
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"strings"
)

func ServePropertyDetails(w http.ResponseWriter, r *http.Request, db PropertyReader) {
	servePropertyDetails(w, r, db, false)
}

func ServeBrowsePropertyDetails(w http.ResponseWriter, r *http.Request, db PropertyReader) {
	servePropertyDetails(w, r, db, true)
}

func servePropertyDetails(w http.ResponseWriter, r *http.Request, db PropertyReader, browse bool) {
	if r.Method != http.MethodGet && (browse || r.Method != http.MethodPost) {
		allow := "GET, POST"
		if browse {
			allow = "GET"
		}
		w.Header().Set("Allow", allow)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uid := strings.TrimSpace(r.URL.Query().Get("uid"))
	// Compatibility for the earlier POST body; new clients use GET ?uid=...
	if uid == "" && r.Method == http.MethodPost {
		var request struct {
			PropertyUID string `json:"PropertyUid"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			http.Error(w, "invalid property request", http.StatusBadRequest)
			return
		}
		uid = strings.TrimSpace(request.PropertyUID)
	}
	if uid == "" || len(uid) > 255 {
		http.Error(w, "uid is required and must be at most 255 bytes", http.StatusBadRequest)
		return
	}
	userUID, ok := r.Context().Value("user_uid").(string)
	if !ok || userUID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var detail PropertyDetails
	var err error
	if browse {
		detail, err = ViewBrowseProperty(db, r.Context(), uid)
	} else {
		user := User{Uid: userUID}
		detail, err = user.ViewProperty(db, r.Context(), uid)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		message := "property not found in portfolio"
		if browse {
			message = "property not found"
		}
		http.Error(w, message, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("failed to load property details: %v", err)
		http.Error(w, "failed to load property details", http.StatusInternalServerError)
		return
	}
	var response any = detail
	if browse {
		// Override only the image tag; keep the portfolio detail contract intact.
		response = struct {
			PropertyDetails
			Image *string `json:"image"`
		}{detail, detail.Image}
	}
	body, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "failed to encode property details", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append(body, '\n'))
}
