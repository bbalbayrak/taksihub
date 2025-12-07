package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Location struct {
	Lat float64 `json:"lat" bson:"lat"`
	Lon float64 `json:"lon" bson:"lon"`
}

type Driver struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	FirstName string             `json:"first_name" bson:"first_name"`
	LastName  string             `json:"last_name" bson:"last_name"`
	Plate     string             `json:"plate" bson:"plate"`
	TaxiType  string             `json:"taxi_type" bson:"taxi_type"`
	CarBrand  string             `json:"car_brand" bson:"car_brand"`
	CarModel  string             `json:"car_model" bson:"car_model"`
	Location  Location           `json:"location" bson:"location"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

const (
	TaxiTypeSari    = "sari"
	TaxiTypeTurkuaz = "turkuaz"
	TaxiTypeSiyah   = "siyah"
)

func IsValidTaxiType(taxiType string) bool {
	switch taxiType {
	case TaxiTypeSari, TaxiTypeTurkuaz, TaxiTypeSiyah:
		return true
	default:
		return false
	}
}
