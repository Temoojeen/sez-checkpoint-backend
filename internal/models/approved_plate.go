package models

import "time"

type ApprovedPlate struct {
	ID               string     `json:"id"`
	PlateNumber      string     `json:"plateNumber"`
	VehicleBrand     string     `json:"vehicleBrand"`
	VehicleModel     string     `json:"vehicleModel"`
	VehicleColor     string     `json:"vehicleColor"`
	ContractID       *string    `json:"contractId"`
	OrganizationID   *string    `json:"organizationId"`
	OrganizationName string     `json:"organizationName,omitempty"`
	ListID           string     `json:"listId"`
	ListName         string     `json:"listName,omitempty"`
	ListType         string     `json:"listType,omitempty"`
	ListColor        string     `json:"listColor,omitempty"`
	ApplicationID    *string    `json:"applicationId,omitempty"` // Добавляем поле
	ApprovedBy       *string    `json:"approvedBy"`
	ApprovedByName   string     `json:"approvedByName,omitempty"`
	ValidFrom        *time.Time `json:"validFrom"`
	ValidUntil       *time.Time `json:"validUntil"`
	IsActive         bool       `json:"isActive"`
	Notes            string     `json:"notes"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type CreateApprovedPlateRequest struct {
	PlateNumber    string `json:"plateNumber" binding:"required"`
	VehicleBrand   string `json:"vehicleBrand"`
	VehicleModel   string `json:"vehicleModel"`
	VehicleColor   string `json:"vehicleColor"`
	OrganizationID string `json:"organizationId" binding:"required"`
	ListID         string `json:"listId" binding:"required"`
	ValidFrom      string `json:"validFrom"`
	ValidUntil     string `json:"validUntil"`
	Notes          string `json:"notes"`
}
