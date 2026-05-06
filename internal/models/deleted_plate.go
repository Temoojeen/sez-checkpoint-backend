package models

import "time"

type DeletedPlate struct {
	ID               string    `json:"id"`
	PlateNumber      string    `json:"plateNumber"`
	VehicleBrand     string    `json:"vehicleBrand,omitempty"`
	VehicleModel     string    `json:"vehicleModel,omitempty"`
	VehicleColor     string    `json:"vehicleColor,omitempty"`
	OrganizationID   *string   `json:"organizationId,omitempty"`
	OrganizationName string    `json:"organizationName,omitempty"`
	ListID           string    `json:"listId"`
	ListName         string    `json:"listName,omitempty"`
	DeletedBy        *string   `json:"deletedBy,omitempty"`
	DeletedByName    string    `json:"deletedByName,omitempty"`
	DeleteReason     string    `json:"deleteReason,omitempty"`
	OriginalPlateID  *string   `json:"originalPlateId,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	DeletedAt        time.Time `json:"deletedAt"`
}
