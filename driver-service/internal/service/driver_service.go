package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/taxihub/driver-service/internal/models"
	"github.com/taxihub/driver-service/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DriverService interface {
	CreateDriver(ctx context.Context, req *models.CreateDriverRequest) (string, error)
	UpdateDriver(ctx context.Context, id string, req *models.UpdateDriverRequest) error
	GetDriverByID(ctx context.Context, id string) (*models.Driver, error)
	ListDrivers(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)
	FindNearbyDrivers(ctx context.Context, lat, lon float64, taxiType string) ([]models.DriverWithDistance, error)
	UpdateDriverLocation(ctx context.Context, id string, req *models.UpdateLocationRequest) error
	DeleteDriver(ctx context.Context, id string) error
	GetDriverByPlate(ctx context.Context, plate string) (*models.Driver, error)
}

type PaginatedResponse struct {
	Data       []models.Driver `json:"data"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalCount int64           `json:"total_count"`
	TotalPages int             `json:"total_pages"`
}

type driverService struct {
	driverRepo repository.DriverRepository
}

func NewDriverService(driverRepo repository.DriverRepository) DriverService {
	return &driverService{
		driverRepo: driverRepo,
	}
}

func (s *driverService) CreateDriver(ctx context.Context, req *models.CreateDriverRequest) (string, error) {
	if req == nil {
		return "", errors.New("request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	driver := &models.Driver{
		ID:        primitive.NewObjectID(),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Plate:     req.Plate,
		TaxiType:  req.TaxiType,
		CarBrand:  req.CarBrand,
		CarModel:  req.CarModel,
		Location: models.Location{
			Lat: req.Lat,
			Lon: req.Lon,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	driverID, err := s.driverRepo.Create(ctx, driver)
	if err != nil {
		return "", fmt.Errorf("failed to create driver: %w", err)
	}

	return driverID, nil
}

func (s *driverService) UpdateDriver(ctx context.Context, id string, req *models.UpdateDriverRequest) error {
	if id == "" {
		return errors.New("driver ID cannot be empty")
	}
	if req == nil {
		return errors.New("request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	existingDriver, err := s.driverRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return fmt.Errorf("driver with ID %s not found", id)
		}
		return fmt.Errorf("failed to find driver: %w", err)
	}

	if req.FirstName != nil {
		existingDriver.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		existingDriver.LastName = *req.LastName
	}
	if req.TaxiType != nil {
		existingDriver.TaxiType = *req.TaxiType
	}
	if req.CarBrand != nil {
		existingDriver.CarBrand = *req.CarBrand
	}
	if req.CarModel != nil {
		existingDriver.CarModel = *req.CarModel
	}
	if req.Lat != nil && req.Lon != nil {
		existingDriver.Location = models.Location{
			Lat: *req.Lat,
			Lon: *req.Lon,
		}
	}

	existingDriver.UpdatedAt = time.Now()

	if err := s.driverRepo.Update(ctx, id, existingDriver); err != nil {
		return fmt.Errorf("failed to update driver: %w", err)
	}

	return nil
}

func (s *driverService) GetDriverByID(ctx context.Context, id string) (*models.Driver, error) {
	if id == "" {
		return nil, errors.New("driver ID cannot be empty")
	}

	driver, err := s.driverRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return nil, fmt.Errorf("driver with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	return driver, nil
}

func (s *driverService) ListDrivers(ctx context.Context, page, pageSize int) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	drivers, totalCount, err := s.driverRepo.FindAll(ctx, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list drivers: %w", err)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	response := &PaginatedResponse{
		Data:       drivers,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}

	return response, nil
}

func (s *driverService) FindNearbyDrivers(ctx context.Context, lat, lon float64, taxiType string) ([]models.DriverWithDistance, error) {
	if lat < -90 || lat > 90 {
		return nil, errors.New("invalid latitude: must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return nil, errors.New("invalid longitude: must be between -180 and 180")
	}

	if taxiType != "" && !models.IsValidTaxiType(taxiType) {
		return nil, fmt.Errorf("invalid taxi type: %s (must be one of: sari, turkuaz, siyah)", taxiType)
	}

	radiusKm := 5.0

	drivers, err := s.driverRepo.FindNearby(ctx, lat, lon, radiusKm, taxiType)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	return drivers, nil
}

func (s *driverService) UpdateDriverLocation(ctx context.Context, id string, req *models.UpdateLocationRequest) error {
	if id == "" {
		return errors.New("driver ID cannot be empty")
	}
	if req == nil {
		return errors.New("request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	existingDriver, err := s.driverRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return fmt.Errorf("driver with ID %s not found", id)
		}
		return fmt.Errorf("failed to find driver: %w", err)
	}

	existingDriver.Location = models.Location{
		Lat: req.Lat,
		Lon: req.Lon,
	}
	existingDriver.UpdatedAt = time.Now()

	if err := s.driverRepo.Update(ctx, id, existingDriver); err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	return nil
}

func (s *driverService) DeleteDriver(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("driver ID cannot be empty")
	}

	_, err := s.driverRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return fmt.Errorf("driver with ID %s not found", id)
		}
		return fmt.Errorf("failed to find driver: %w", err)
	}

	if err := s.driverRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete driver: %w", err)
	}

	return nil
}

func (s *driverService) GetDriverByPlate(ctx context.Context, plate string) (*models.Driver, error) {
	if plate == "" {
		return nil, errors.New("plate cannot be empty")
	}

	driver, err := s.driverRepo.FindByPlate(ctx, plate)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return nil, fmt.Errorf("driver with plate %s not found", plate)
		}
		return nil, fmt.Errorf("failed to get driver by plate: %w", err)
	}

	return driver, nil
}
