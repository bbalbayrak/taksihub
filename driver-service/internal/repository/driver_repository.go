package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/taxihub/driver-service/internal/config"
	"github.com/taxihub/driver-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DriverRepository interface {
	Create(ctx context.Context, driver *models.Driver) (string, error)
	Update(ctx context.Context, id string, driver *models.Driver) error
	FindByID(ctx context.Context, id string) (*models.Driver, error)
	FindAll(ctx context.Context, page, pageSize int) ([]models.Driver, int64, error)
	FindNearby(ctx context.Context, lat, lon, radiusKm float64, taxiType string) ([]models.DriverWithDistance, error)
	FindByPlate(ctx context.Context, plate string) (*models.Driver, error)
	Delete(ctx context.Context, id string) error
}

type MongoDriverRepository struct {
	collection *mongo.Collection
}

func NewMongoDriverRepository(db *config.MongoDB) *MongoDriverRepository {
	return &MongoDriverRepository{
		collection: db.GetCollection("drivers"),
	}
}

func (r *MongoDriverRepository) Create(ctx context.Context, driver *models.Driver) (string, error) {
	if driver == nil {
		return "", errors.New("driver cannot be nil")
	}

	now := time.Now()
	driver.CreatedAt = now
	driver.UpdatedAt = now

	if driver.ID.IsZero() {
		driver.ID = primitive.NewObjectID()
	}

	result, err := r.collection.InsertOne(ctx, driver)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("driver with plate %s already exists", driver.Plate)
		}
		return "", fmt.Errorf("failed to create driver: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		return oid.Hex(), nil
	}

	return driver.ID.Hex(), nil
}

func (r *MongoDriverRepository) Update(ctx context.Context, id string, driver *models.Driver) error {
	if id == "" {
		return errors.New("driver ID cannot be empty")
	}
	if driver == nil {
		return errors.New("driver cannot be nil")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid driver ID format: %w", err)
	}

	driver.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"first_name": driver.FirstName,
			"last_name":  driver.LastName,
			"plate":      driver.Plate,
			"taxi_type":  driver.TaxiType,
			"car_brand":  driver.CarBrand,
			"car_model":  driver.CarModel,
			"location":   driver.Location,
			"updated_at": driver.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("driver with plate %s already exists", driver.Plate)
		}
		return fmt.Errorf("failed to update driver: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("driver with ID %s not found", id)
	}

	return nil
}

func (r *MongoDriverRepository) FindByID(ctx context.Context, id string) (*models.Driver, error) {
	if id == "" {
		return nil, errors.New("driver ID cannot be empty")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid driver ID format: %w", err)
	}

	var driver models.Driver
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&driver)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("driver with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to find driver: %w", err)
	}

	return &driver, nil
}

func (r *MongoDriverRepository) FindAll(ctx context.Context, page, pageSize int) ([]models.Driver, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	skip := (page - 1) * pageSize

	totalCount, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count drivers: %w", err)
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSort(bson.M{"created_at": -1}) // Sort by creation date, newest first

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find drivers: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var drivers []models.Driver
	if err = cursor.All(ctx, &drivers); err != nil {
		return nil, 0, fmt.Errorf("failed to decode drivers: %w", err)
	}

	return drivers, totalCount, nil
}

func (r *MongoDriverRepository) FindNearby(ctx context.Context, lat, lon, radiusKm float64, taxiType string) ([]models.DriverWithDistance, error) {
	if lat < -90 || lat > 90 {
		return nil, errors.New("invalid latitude value")
	}
	if lon < -180 || lon > 180 {
		return nil, errors.New("invalid longitude value")
	}
	if radiusKm <= 0 {
		return nil, errors.New("radius must be positive")
	}

	center := bson.M{
		"type":        "Point",
		"coordinates": []float64{lon, lat},
	}

	query := bson.M{
		"location": bson.M{
			"$nearSphere": bson.M{
				"$geometry":    center,
				"$maxDistance": radiusKm * 1000,
			},
		},
	}

	if taxiType != "" && models.IsValidTaxiType(taxiType) {
		query["taxi_type"] = taxiType
	}

	pipeline := []bson.M{
		{
			"$geoNear": bson.M{
				"near":          center,
				"distanceField": "distance",
				"maxDistance":   radiusKm * 1000,
				"spherical":     true,
			},
		},
	}

	if taxiType != "" && models.IsValidTaxiType(taxiType) {
		pipeline = append(pipeline, bson.M{
			"$match": bson.M{"taxi_type": taxiType},
		})
	}

	pipeline = append(pipeline, bson.M{"$limit": 50})

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		models.Driver `bson:",inline"`
		Distance      float64 `bson:"distance"`
	}

	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode nearby drivers: %w", err)
	}

	driversWithDistance := make([]models.DriverWithDistance, len(results))
	for i, result := range results {
		driversWithDistance[i] = models.DriverWithDistance{
			Driver:     result.Driver,
			DistanceKm: result.Distance / 1000,
		}
	}

	return driversWithDistance, nil
}

func (r *MongoDriverRepository) FindByPlate(ctx context.Context, plate string) (*models.Driver, error) {
	if plate == "" {
		return nil, errors.New("plate cannot be empty")
	}

	var driver models.Driver
	err := r.collection.FindOne(ctx, bson.M{"plate": plate}).Decode(&driver)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDriverNotFound
		}
		return nil, fmt.Errorf("failed to find driver by plate: %w", err)
	}

	return &driver, nil
}

func (r *MongoDriverRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("driver ID cannot be empty")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid driver ID format: %w", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete driver: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("driver with ID %s not found", id)
	}

	return nil
}
