package UserActions

import (
	p "Ptixiaki/Backend/Api"
	Gd "Ptixiaki/Backend/ExternalServices/Geo"
	explain "Ptixiaki/Worker/explainability"
	t "Ptixiaki/Worker/types"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
)

type User struct {
	Uid     string
	IsAdmin bool
}

func (u *User) AddProperty(db *pgxpool.Pool, ctx context.Context, prop p.PropertyFeatures, treeAttr, tree *t.ModelPredictor, transaction string) (string, error) {
	if err := explain.ValidateTransaction(transaction); err != nil {
		return "", err
	}
	if err := p.ValidatePropertyInput(prop); err != nil {
		return "", err
	}
	geo := Gd.Geo{Db: db, Ctx: ctx}
	geoData, err := Gd.Geocoder.Resolve(ctx, prop.AddressOSMType, prop.AddressOSMID, prop.Address)
	if err != nil {
		return "", err
	}
	prop.Address = geoData.DisplayName
	zoneValue, err := geo.GetZoneValue(geoData.Lat, geoData.Lon)
	if err != nil {
		return "", err
	}
	propertyUID, err := InsertProperty(db, ctx, prop, geoData, zoneValue, u.Uid, treeAttr, tree, transaction)
	if err != nil {
		log.Println("Error inserting property:", err)
	}
	return propertyUID, err
}
