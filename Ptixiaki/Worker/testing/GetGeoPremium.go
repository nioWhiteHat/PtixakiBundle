package testing

import (
	"fmt"
	"log"
	"math"
)

type geoPremMinimal struct {
	UID              string  `json:"UID"`
	GeoPremium       float64 `json:"GeoPremium"`
	GeoPremiumZScore float64 `json:"GeoPremiumZScore"`
}

// Internal struct just to hold both values in our map
type geoPremiumData struct {
	Premium float64
	ZScore  float64
}

func (c *Calculator) GetGeoPremium(uid string) (float64, float64, error) {
	query := `WITH target AS (
		SELECT p.transaction_type, p.item_type, p.subtype, pg.geom
		FROM properties p
		JOIN property_geo_data pg ON p.uid = pg.property_uid
		WHERE p.uid = $1
	)
	SELECT (e.asking_price_per_sqm - e.attr_price) / NULLIF(e.asking_price_per_sqm, 0) * 100
	FROM properties p
	JOIN property_geo_data pg ON p.uid = pg.property_uid
	JOIN estimation_data e ON p.uid = e.uid
	CROSS JOIN target t
	WHERE p.uid != $1
	  AND p.transaction_type = t.transaction_type
	  AND p.item_type = t.item_type
	  AND (
		(t.item_type = 'residence'
		 AND t.subtype IN ('apartment', 'maisonette', 'detached_house')
		 AND p.subtype IN ('apartment', 'maisonette', 'detached_house'))
		OR
		(NOT (t.item_type = 'residence' AND t.subtype IN ('apartment', 'maisonette', 'detached_house'))
		 AND p.subtype IS NOT DISTINCT FROM t.subtype)
	  )
	  AND e.asking_price_per_sqm IS NOT NULL
	  AND e.asking_price_per_sqm > 0
	  AND e.attr_price IS NOT NULL
	  AND ST_DWithin(pg.geom::geography, t.geom::geography, 300)
	ORDER BY pg.geom::geography <-> t.geom::geography
	LIMIT 50;`
	rows, err := c.Db.Query(c.Ctx, query, uid)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get geo premium neighbors for uid %s: %v", uid, err)
	}
	defer rows.Close()

	var residuals []float64
	var sum float64
	for rows.Next() {
		var residual float64
		if err := rows.Scan(&residual); err != nil {
			return 0, 0, fmt.Errorf("failed to scan geo premium neighbor for uid %s: %v", uid, err)
		}
		residuals = append(residuals, residual)
		sum += residual
	}
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("failed to read geo premium neighbors for uid %s: %v", uid, err)
	}
	if len(residuals) == 0 {
		return 0, 0, fmt.Errorf("no eligible geo premium neighbors found for uid %s", uid)
	}

	geoPremium := sum / float64(len(residuals))
	var squaredDiffSum float64
	for _, residual := range residuals {
		diff := residual - geoPremium
		squaredDiffSum += diff * diff
	}
	geoStd := math.Sqrt(squaredDiffSum / float64(len(residuals)))

	log.Println("***********************Calculated geo premium and std for uid:", uid, "NeighborCount:", len(residuals), "GeoPremium:", geoPremium, "GeoStd:", geoStd)
	return geoPremium, geoStd, nil
}
