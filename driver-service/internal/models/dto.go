package models

import (
	"fmt"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TurkishLicensePlateValidator(fl validator.FieldLevel) bool {
	plate := fl.Field().String()

	plateNoSpace := regexp.MustCompile(`\s+`).ReplaceAllString(plate, "")

	pattern := `^[0-9]{2}[A-Za-z]{1,3}[0-9]{1,4}$`
	matched, _ := regexp.MatchString(pattern, plateNoSpace)

	return matched
}

type CreateDriverRequest struct {
	FirstName string  `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string  `json:"last_name" validate:"required,min=2,max=50"`
	Plate     string  `json:"plate" validate:"required,turkish_plate"`
	TaxiType  string  `json:"taxi_type" validate:"required,oneof=sari turkuaz siyah"`
	CarBrand  string  `json:"car_brand" validate:"required,min=2,max=30"`
	CarModel  string  `json:"car_model" validate:"required,min=1,max=30"`
	Lat       float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon       float64 `json:"lon" validate:"required,min=-180,max=180"`
}

func (r *CreateDriverRequest) ToDriver() *Driver {
	return &Driver{
		ID:        primitive.NewObjectID(),
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Plate:     r.Plate,
		TaxiType:  r.TaxiType,
		CarBrand:  r.CarBrand,
		CarModel:  r.CarModel,
		Location: Location{
			Lat: r.Lat,
			Lon: r.Lon,
		},
	}
}

func (r *CreateDriverRequest) Validate() error {
	validate := validator.New()

	validate.RegisterValidation("turkish_plate", TurkishLicensePlateValidator)

	return validate.Struct(r)
}

type UpdateDriverRequest struct {
	FirstName *string  `json:"first_name,omitempty" validate:"omitempty,min=2,max=50"`
	LastName  *string  `json:"last_name,omitempty" validate:"omitempty,min=2,max=50"`
	TaxiType  *string  `json:"taxi_type,omitempty" validate:"omitempty,oneof=sari turkuaz siyah"`
	CarBrand  *string  `json:"car_brand,omitempty" validate:"omitempty,min=2,max=30"`
	CarModel  *string  `json:"car_model,omitempty" validate:"omitempty,min=1,max=30"`
	Lat       *float64 `json:"lat,omitempty" validate:"omitempty,min=-90,max=90"`
	Lon       *float64 `json:"lon,omitempty" validate:"omitempty,min=-180,max=180"`
}

func (r *UpdateDriverRequest) HasLocation() bool {
	return r.Lat != nil && r.Lon != nil
}

func (r *UpdateDriverRequest) GetLocation() *Location {
	if r.HasLocation() {
		return &Location{
			Lat: *r.Lat,
			Lon: *r.Lon,
		}
	}
	return nil
}

func (r *UpdateDriverRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon float64 `json:"lon" validate:"required,min=-180,max=180"`
}

func (r *UpdateLocationRequest) ToLocation() Location {
	return Location{
		Lat: r.Lat,
		Lon: r.Lon,
	}
}

func (r *UpdateLocationRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

type DriverResponse struct {
	ID        string   `json:"id"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Plate     string   `json:"plate"`
	TaxiType  string   `json:"taxi_type"`
	CarBrand  string   `json:"car_brand"`
	CarModel  string   `json:"car_model"`
	Location  Location `json:"location"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func NewDriverResponse(driver *Driver) *DriverResponse {
	return &DriverResponse{
		ID:        driver.ID.Hex(),
		FirstName: driver.FirstName,
		LastName:  driver.LastName,
		Plate:     driver.Plate,
		TaxiType:  driver.TaxiType,
		CarBrand:  driver.CarBrand,
		CarModel:  driver.CarModel,
		Location:  driver.Location,
		CreatedAt: driver.CreatedAt.Format(time.RFC3339),
		UpdatedAt: driver.UpdatedAt.Format(time.RFC3339),
	}
}

type DriverWithDistanceResponse struct {
	ID         string   `json:"id"`
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Plate      string   `json:"plate"`
	TaxiType   string   `json:"taxi_type"`
	CarBrand   string   `json:"car_brand"`
	CarModel   string   `json:"car_model"`
	Location   Location `json:"location"`
	DistanceKm float64  `json:"distance_km"`
}

func NewDriverWithDistanceResponse(driver DriverWithDistance) *DriverWithDistanceResponse {
	distance := fmt.Sprintf("%.1f", driver.DistanceKm)
	var roundedDistance float64
	fmt.Sscanf(distance, "%f", &roundedDistance)

	return &DriverWithDistanceResponse{
		ID:         driver.ID.Hex(),
		FirstName:  driver.FirstName,
		LastName:   driver.LastName,
		Plate:      driver.Plate,
		TaxiType:   driver.TaxiType,
		CarBrand:   driver.CarBrand,
		CarModel:   driver.CarModel,
		Location:   driver.Location,
		DistanceKm: roundedDistance,
	}
}

type ErrorResponse struct {
	Error   string   `json:"error"`
	Details []string `json:"details,omitempty"`
	Code    int      `json:"code,omitempty"`
}

func NewErrorResponse(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: message,
	}
}

func NewValidationErrorResponse(message string, details []string) *ErrorResponse {
	return &ErrorResponse{
		Error:   message,
		Details: details,
	}
}

func NewErrorResponseWithCode(message string, code int) *ErrorResponse {
	return &ErrorResponse{
		Error: message,
		Code:  code,
	}
}

type ListDriversResponse struct {
	Data       []DriverResponse `json:"data"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalCount int64            `json:"total_count"`
	TotalPages int              `json:"total_pages"`
}

func NewListDriversResponse(serviceResp *PaginatedServiceResponse) *ListDriversResponse {
	drivers := make([]DriverResponse, len(serviceResp.Data))
	for i, driver := range serviceResp.Data {
		drivers[i] = *NewDriverResponse(&driver)
	}

	return &ListDriversResponse{
		Data:       drivers,
		Page:       serviceResp.Page,
		PageSize:   serviceResp.PageSize,
		TotalCount: serviceResp.TotalCount,
		TotalPages: serviceResp.TotalPages,
	}
}

type PaginatedServiceResponse struct {
	Data       []Driver `json:"data"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	TotalCount int64    `json:"total_count"`
	TotalPages int      `json:"total_pages"`
}

type DriverWithDistance struct {
	Driver
	DistanceKm float64 `json:"distance_km"`
}
