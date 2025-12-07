package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/taxihub/driver-service/internal/models"
	"github.com/taxihub/driver-service/internal/repository"
	"github.com/taxihub/driver-service/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DriverHandler struct {
	driverService service.DriverService
	validator     *validator.Validate
}

func NewDriverHandler(driverService service.DriverService) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		validator:     validator.New(),
	}
}

func (h *DriverHandler) RegisterRoutes(app *fiber.App) {
	v1 := app.Group("/api/v1")

	drivers := v1.Group("/drivers")
	{
		drivers.Post("/", h.CreateDriver)
		drivers.Get("/", h.ListDrivers)
		drivers.Get("/:id", h.GetDriver)
		drivers.Put("/:id", h.UpdateDriver)
		drivers.Delete("/:id", h.DeleteDriver)
		drivers.Get("/nearby", h.FindNearbyDrivers)
		drivers.Put("/:id/location", h.UpdateDriverLocation)
	}
}

func (h *DriverHandler) CreateDriver(c *fiber.Ctx) error {
	var req models.CreateDriverRequest
	if err := c.BodyParser(&req); err != nil {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid JSON format", nil)
	}

	// Validate requests
	if err := req.Validate(); err != nil {
		var validationErrors []string
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, e := range validationErr {
				validationErrors = append(validationErrors, h.formatValidationError(e))
			}
		} else {
			validationErrors = append(validationErrors, err.Error())
		}
		return h.ErrorResponse(c, http.StatusBadRequest, "Validation failed", validationErrors)
	}

	driverID, err := h.driverService.CreateDriver(c.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrDriverAlreadyExists) {
			return h.ErrorResponse(c, http.StatusConflict, "Driver with this plate already exists", nil)
		}
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to create driver", []string{err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id": driverID,
	})
}

func (h *DriverHandler) UpdateDriver(c *fiber.Ctx) error {
	id := c.Params("id")
	if !h.isValidObjectID(id) {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID format", nil)
	}

	var req models.UpdateDriverRequest
	if err := c.BodyParser(&req); err != nil {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid JSON format", nil)
	}

	// Validate requests
	if err := req.Validate(); err != nil {
		var validationErrors []string
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, e := range validationErr {
				validationErrors = append(validationErrors, h.formatValidationError(e))
			}
		} else {
			validationErrors = append(validationErrors, err.Error())
		}
		return h.ErrorResponse(c, http.StatusBadRequest, "Validation failed", validationErrors)
	}

	if err := h.driverService.UpdateDriver(c.Context(), id, &req); err != nil {
		if errors.Is(err, service.ErrDriverNotFound) {
			return h.ErrorResponse(c, http.StatusNotFound, "Driver not found", nil)
		}
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to update driver", []string{err.Error()})
	}

	driver, err := h.driverService.GetDriverByID(c.Context(), id)
	if err != nil {
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch updated driver", []string{err.Error()})
	}

	return c.JSON(models.NewDriverResponse(driver))
}

func (h *DriverHandler) GetDriver(c *fiber.Ctx) error {
	id := c.Params("id")
	if !h.isValidObjectID(id) {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID format", nil)
	}

	driver, err := h.driverService.GetDriverByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrDriverNotFound) {
			return h.ErrorResponse(c, http.StatusNotFound, "Driver not found", nil)
		}
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to get driver", []string{err.Error()})
	}

	return c.JSON(models.NewDriverResponse(driver))
}

func (h *DriverHandler) ListDrivers(c *fiber.Ctx) error {
	page := 1
	pageSize := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			if ps > 100 {
				pageSize = 100
			} else {
				pageSize = ps
			}
		}
	}

	response, err := h.driverService.ListDrivers(c.Context(), page, pageSize)
	if err != nil {
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to list drivers", []string{err.Error()})
	}

	serviceResp := &models.PaginatedServiceResponse{
		Data:       response.Data,
		Page:       response.Page,
		PageSize:   response.PageSize,
		TotalCount: response.TotalCount,
		TotalPages: response.TotalPages,
	}

	return c.JSON(models.NewListDriversResponse(serviceResp))
}

func (h *DriverHandler) DeleteDriver(c *fiber.Ctx) error {
	id := c.Params("id")
	if !h.isValidObjectID(id) {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID format", nil)
	}

	if err := h.driverService.DeleteDriver(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrDriverNotFound) {
			return h.ErrorResponse(c, http.StatusNotFound, "Driver not found", nil)
		}
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete driver", []string{err.Error()})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

func (h *DriverHandler) FindNearbyDrivers(c *fiber.Ctx) error {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	taxiType := c.Query("taxiType")

	if latStr == "" || lonStr == "" {
		return h.ErrorResponse(c, http.StatusBadRequest, "lat and lon query parameters are required", nil)
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid latitude format", nil)
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid longitude format", nil)
	}

	drivers, err := h.driverService.FindNearbyDrivers(c.Context(), lat, lon, taxiType)
	if err != nil {
		if errors.Is(err, service.ErrInvalidLocation) {
			return h.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		}
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to find nearby drivers", []string{err.Error()})
	}

	response := make([]*models.DriverWithDistanceResponse, len(drivers))
	for i, driver := range drivers {
		response[i] = models.NewDriverWithDistanceResponse(driver)
	}

	return c.JSON(fiber.Map{
		"drivers": response,
		"location": fiber.Map{
			"lat": lat,
			"lon": lon,
		},
	})
}

func (h *DriverHandler) UpdateDriverLocation(c *fiber.Ctx) error {
	id := c.Params("id")
	if !h.isValidObjectID(id) {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID format", nil)
	}

	var req models.UpdateLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid JSON format", nil)
	}

	if err := req.Validate(); err != nil {
		var validationErrors []string
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, e := range validationErr {
				validationErrors = append(validationErrors, h.formatValidationError(e))
			}
		} else {
			validationErrors = append(validationErrors, err.Error())
		}
		return h.ErrorResponse(c, http.StatusBadRequest, "Validation failed", validationErrors)
	}

	if err := h.driverService.UpdateDriverLocation(c.Context(), id, &req); err != nil {
		if errors.Is(err, service.ErrDriverNotFound) {
			return h.ErrorResponse(c, http.StatusNotFound, "Driver not found", nil)
		}
		return h.ErrorResponse(c, http.StatusInternalServerError, "Failed to update driver location", []string{err.Error()})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Location updated successfully",
	})
}

func (h *DriverHandler) isValidObjectID(id string) bool {
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}

func (h *DriverHandler) ErrorResponse(c *fiber.Ctx, statusCode int, message string, details []string) error {
	response := models.ErrorResponse{
		Error:   message,
		Details: details,
		Code:    statusCode,
	}
	return c.Status(statusCode).JSON(response)
}

func (h *DriverHandler) formatValidationError(err validator.FieldError) string {
	field := strings.ToLower(err.Field())
	tag := err.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, err.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "turkish_plate":
		return "plate must be a valid Turkish license plate (e.g., 34 ABC 123)"
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

func (h *DriverHandler) HandleValidationErrors(err error) []string {
	var errors []string
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErr {
			errors = append(errors, h.formatValidationError(e))
		}
	} else {
		errors = append(errors, err.Error())
	}
	return errors
}

func (h *DriverHandler) HandleServiceErrors(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrDriverNotFound):
		return h.ErrorResponse(c, http.StatusNotFound, "Driver not found", nil)
	case errors.Is(err, service.ErrDriverAlreadyExists):
		return h.ErrorResponse(c, http.StatusConflict, "Driver already exists", nil)
	case errors.Is(err, service.ErrInvalidID):
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID", nil)
	case errors.Is(err, service.ErrValidationFailed):
		return h.ErrorResponse(c, http.StatusBadRequest, "Validation failed", nil)
	case errors.Is(err, repository.ErrDriverNotFound):
		return h.ErrorResponse(c, http.StatusNotFound, "Driver not found", nil)
	case errors.Is(err, repository.ErrInvalidID):
		return h.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID format", nil)
	default:
		return h.ErrorResponse(c, http.StatusInternalServerError, "Internal server error", []string{err.Error()})
	}
}
