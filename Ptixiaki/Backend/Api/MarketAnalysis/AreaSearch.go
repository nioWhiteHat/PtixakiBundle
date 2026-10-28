package marketanalysis

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/unicode/norm"
)

type AreaCatalog struct {
	mu       sync.Mutex
	loadedAt time.Time
	areas    []searchArea
}

type searchArea struct {
	NominatimResponse
	name  string
	level int
}

func (catalog *AreaCatalog) Search(ctx context.Context, db *pgxpool.Pool, query string) ([]NominatimResponse, error) {
	query = areaSearchKey(query)
	if len([]rune(query)) < 2 {
		return []NominatimResponse{}, nil
	}
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if time.Since(catalog.loadedAt) > time.Hour {
		rows, err := db.Query(ctx, `SELECT name, admin_level,
		 ST_YMin(Box3D(geom)), ST_YMax(Box3D(geom)), ST_XMin(Box3D(geom)), ST_XMax(Box3D(geom))
		 FROM admin_boundaries WHERE admin_level BETWEEN 6 AND 10 AND NOT ST_IsEmpty(geom)
		 ORDER BY name, admin_level`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		areas := []searchArea{}
		for rows.Next() {
			var name string
			var level int
			var south, north, west, east float64
			if err := rows.Scan(&name, &level, &south, &north, &west, &east); err != nil {
				return nil, err
			}
			if south >= north || west >= east {
				continue
			}
			area := searchArea{name: areaSearchKey(name), level: level}
			area.DisplayName = name + ", Greece"
			area.Lat = formatCoordinate((south + north) / 2)
			area.Lon = formatCoordinate((west + east) / 2)
			area.BoundingBox = []string{
				formatCoordinate(south),
				formatCoordinate(north),
				formatCoordinate(west),
				formatCoordinate(east),
			}
			areas = append(areas, area)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		catalog.areas = areas
		catalog.loadedAt = time.Now()
	}
	return matchAreas(catalog.areas, query), nil
}

func matchAreas(areas []searchArea, query string) []NominatimResponse {
	matches := []searchArea{}
	for _, area := range areas {
		if strings.Contains(area.name, query) {
			matches = append(matches, area)
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return areaRank(matches[i].level) < areaRank(matches[j].level)
	})
	results := []NominatimResponse{}
	for _, area := range matches {
		results = append(results, area.NominatimResponse)
		if len(results) == 8 {
			break
		}
	}
	return results
}

func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func areaRank(level int) int {
	if level == 7 || level == 8 {
		return 0
	}
	if level == 6 {
		return 1
	}
	return 2
}

func areaSearchKey(value string) string {
	var clean strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(strings.TrimSpace(value))) {
		if !unicode.Is(unicode.Mn, r) {
			clean.WriteRune(r)
		}
	}
	value = strings.NewReplacer("αι", "e", "ει", "i", "οι", "i", "ου", "ou",
		"α", "a", "β", "v", "γ", "g", "δ", "d", "ε", "e", "ζ", "z", "η", "i", "θ", "th",
		"ι", "i", "κ", "k", "λ", "l", "μ", "m", "ν", "n", "ξ", "x", "ο", "o", "π", "p",
		"ρ", "r", "σ", "s", "ς", "s", "τ", "t", "υ", "y", "φ", "f", "χ", "ch", "ψ", "ps", "ω", "o").Replace(clean.String())
	aliases := [][2]string{
		{"piraeus", "pire"},
		{"pireaus", "pire"},
		{"peiraias", "pire"},
		{"piraias", "pire"},
		{"piraios", "pire"},
		{"athens", "athin"},
		{"patras", "patr"},
		{"heraklion", "irakl"},
	}
	for _, alias := range aliases {
		if len(value) >= 3 && strings.HasPrefix(alias[0], value) {
			return alias[1]
		}
	}
	return value
}
