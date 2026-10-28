package UserActions

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PortfolioPropertyOverview struct {
	UID                  string   `json:"uid"`
	EstimatedPricePerSqm *float64 `json:"estimated_price_per_sqm"`
	AskingPricePerSqm    *float64 `json:"asking_price_per_sqm"`
	SquareMeters         *int     `json:"square_meters"`
	Address              *string  `json:"address"`
	Image                *string  `json:"image,omitempty"`
	Elo                  *int     `json:"elo"`
	GeoPremium           *float64 `json:"geo_premium"`
}

type PortfolioResponse struct {
	PropertyCount       int                         `json:"property_count"`
	TotalEstimatedValue float64                     `json:"total_estimated_value"`
	Properties          []PortfolioPropertyOverview `json:"properties"`
}

func (u *User) GetPortfolio(db *pgxpool.Pool, ctx context.Context) (PortfolioResponse, error) {
	response := PortfolioResponse{
		Properties: make([]PortfolioPropertyOverview, 0),
	}

	query := `
		SELECT
			p.uid,
			ed.predicted_price_per_sqm,
			ed.asking_price_per_sqm,
			ba.square_meters,
			pgd.address_text,
			pe.elo,
			ed.geo_premium
		FROM user_properties up
		JOIN properties p ON p.uid = up.property_uid
		LEFT JOIN base_attributes ba ON ba.property_uid = p.uid
		LEFT JOIN property_geo_data pgd ON pgd.property_uid = p.uid
		LEFT JOIN estimation_data ed ON ed.uid = p.uid
		LEFT JOIN properties_elo pe ON pe.property_uid = p.uid
		WHERE up.user_uid = $1
		ORDER BY up.created_at DESC, p.uid
	`

	rows, err := db.Query(ctx, query, u.Uid)
	if err != nil {
		return PortfolioResponse{}, fmt.Errorf("query user portfolio: %w", err)
	}
	defer rows.Close()

	propertyUIDs := make([]string, 0)
	for rows.Next() {
		var property PortfolioPropertyOverview
		var squareMeters *int32
		var elo *int32

		if err := rows.Scan(
			&property.UID,
			&property.EstimatedPricePerSqm,
			&property.AskingPricePerSqm,
			&squareMeters,
			&property.Address,
			&elo,
			&property.GeoPremium,
		); err != nil {
			return PortfolioResponse{}, fmt.Errorf("scan user portfolio property: %w", err)
		}

		if squareMeters != nil {
			value := int(*squareMeters)
			property.SquareMeters = &value
		}
		if elo != nil {
			value := int(*elo)
			property.Elo = &value
		}
		if property.EstimatedPricePerSqm != nil && property.SquareMeters != nil {
			response.TotalEstimatedValue += *property.EstimatedPricePerSqm * float64(*property.SquareMeters)
		}

		propertyUIDs = append(propertyUIDs, property.UID)
		response.Properties = append(response.Properties, property)
	}
	if err := rows.Err(); err != nil {
		return PortfolioResponse{}, fmt.Errorf("read user portfolio properties: %w", err)
	}

	images, err := loadPrimaryPortfolioImages(ctx, db, propertyUIDs)
	if err != nil {
		return PortfolioResponse{}, err
	}
	for i := range response.Properties {
		response.Properties[i].Image = images[response.Properties[i].UID]
	}

	response.PropertyCount = len(response.Properties)
	return response, nil
}

func loadPrimaryPortfolioImages(ctx context.Context, db propertyQueryer, propertyUIDs []string) (map[string]*string, error) {
	images := make(map[string]*string, len(propertyUIDs))
	if len(propertyUIDs) == 0 {
		return images, nil
	}
	// Avoid aborting a read-only detail transaction if migration 001 is absent.
	var installed bool
	if err := db.QueryRow(ctx, `SELECT to_regclass('public.property_images') IS NOT NULL`).Scan(&installed); err != nil {
		return nil, fmt.Errorf("check property image storage: %w", err)
	}
	if !installed {
		return images, nil
	}

	query := `
		SELECT DISTINCT ON (property_uid)
			property_uid, image_url, image_data, mime_type
		FROM property_images
		WHERE property_uid = ANY($1)
		ORDER BY property_uid, position, created_at
	`
	rows, err := db.Query(ctx, query, propertyUIDs)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			log.Printf("property_images table is not installed; returning portfolio images as null")
			return images, nil
		}
		return nil, fmt.Errorf("query portfolio images: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var propertyUID string
		var imageURL *string
		var imageData []byte
		var mimeType *string
		if err := rows.Scan(&propertyUID, &imageURL, &imageData, &mimeType); err != nil {
			return nil, fmt.Errorf("scan portfolio image: %w", err)
		}
		images[propertyUID] = portfolioImageSource(imageURL, imageData, mimeType)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read portfolio images: %w", err)
	}
	return images, nil
}

func portfolioImageSource(imageURL *string, imageData []byte, mimeType *string) *string {
	if imageURL != nil && strings.TrimSpace(*imageURL) != "" {
		value := strings.TrimSpace(*imageURL)
		return &value
	}
	if len(imageData) == 0 {
		return nil
	}

	mime := ""
	if mimeType != nil {
		mime = strings.TrimSpace(*mimeType)
	}
	if mime == "" {
		mime = http.DetectContentType(imageData)
	}
	value := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(imageData)
	return &value
}
