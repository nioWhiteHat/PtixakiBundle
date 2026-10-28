package marketanalysis
import (
	t "Ptixiaki/Worker/types"
)
type ResidentialProperty struct {
	t.Global
	t.Residential
	t.ResidentialPropertySpecifics
	GeoPremium float64 `db:"geo_premium"`
}
type ModeledResidentialProperty struct {
	t.ResidentialFeatures
	GeoPremium float64
}