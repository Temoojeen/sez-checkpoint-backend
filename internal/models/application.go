package models

import "time"

type Application struct {
	ID                   string     `json:"id"`
	PlateNumber          string     `json:"plateNumber"`
	VehicleBrand         string     `json:"vehicleBrand"`
	VehicleModel         string     `json:"vehicleModel"`
	VehicleColor         string     `json:"vehicleColor"`
	ContractID           *string    `json:"contractId"`
	ContractNumber       string     `json:"contractNumber,omitempty"`
	OrganizationID       *string    `json:"organizationId"`
	OrganizationName     string     `json:"organizationName,omitempty"`
	ListID               string     `json:"listId"`
	ListName             string     `json:"listName,omitempty"`
	ApplicantID          string     `json:"applicantId"`
	ApplicantName        string     `json:"applicantName,omitempty"`
	Status               string     `json:"status"`
	OperatorID           *string    `json:"operatorId"`
	SupervisorID         *string    `json:"supervisorId"`
	OperatorApprovedAt   *time.Time `json:"operatorApprovedAt"`
	SupervisorApprovedAt *time.Time `json:"supervisorApprovedAt"`
	RejectedAt           *time.Time `json:"rejectedAt"`
	RejectReason         string     `json:"rejectReason"`
	ValidFrom            *time.Time `json:"validFrom"`
	ValidUntil           *time.Time `json:"validUntil"`
	Notes                string     `json:"notes"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type CreateApplicationRequest struct {
	PlateNumber    string `json:"plateNumber" binding:"required"`
	VehicleBrand   string `json:"vehicleBrand"`
	VehicleModel   string `json:"vehicleModel"`
	VehicleColor   string `json:"vehicleColor"`
	ContractNumber string `json:"contractNumber" binding:"required"`
	ListID         string `json:"listId" binding:"required"`
	ValidFrom      string `json:"validFrom"`
	ValidUntil     string `json:"validUntil"`
	Notes          string `json:"notes"`
}

type ApproveApplicationRequest struct {
	ApplicationID string `json:"applicationId" binding:"required"`
}

type RejectApplicationRequest struct {
	ApplicationID string `json:"applicationId" binding:"required"`
	Reason        string `json:"reason"`
}
