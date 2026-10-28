package UserActions

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestPortfolioImageSourcePrefersURL(t *testing.T) {
	url := " https://example.com/property.jpg "
	mime := "image/png"
	got := portfolioImageSource(&url, []byte("image bytes"), &mime)
	if got == nil || *got != "https://example.com/property.jpg" {
		t.Fatalf("portfolioImageSource() = %v, want URL", got)
	}
}

func TestPortfolioImageSourceBuildsDataURL(t *testing.T) {
	data := []byte("image bytes")
	mime := "image/jpeg"
	got := portfolioImageSource(nil, data, &mime)
	want := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
	if got == nil || *got != want {
		t.Fatalf("portfolioImageSource() = %v, want %q", got, want)
	}
}

func TestPortfolioImageSourceDetectsMimeType(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	got := portfolioImageSource(nil, pngHeader, nil)
	if got == nil || !strings.HasPrefix(*got, "data:image/png;base64,") {
		t.Fatalf("portfolioImageSource() = %v, want PNG data URL", got)
	}
}

func TestPortfolioImageSourceReturnsNilWithoutImage(t *testing.T) {
	if got := portfolioImageSource(nil, nil, nil); got != nil {
		t.Fatalf("portfolioImageSource() = %v, want nil", got)
	}
}

func TestPortfolioPropertyOverviewIncludesUID(t *testing.T) {
	payload, err := json.Marshal(PortfolioPropertyOverview{UID: "property_123"})
	if err != nil {
		t.Fatalf("marshal portfolio property: %v", err)
	}
	if !strings.Contains(string(payload), `"uid":"property_123"`) {
		t.Fatalf("portfolio property JSON %s does not contain uid", payload)
	}
}
