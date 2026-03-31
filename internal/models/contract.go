package models

import "time"

type Contract struct {
	ID               string     `json:"id"`
	ContractNumber   string     `json:"contractNumber"`
	OrganizationID   string     `json:"organizationId"`
	OrganizationName string     `json:"organizationName,omitempty"`
	ContractDate     time.Time  `json:"contractDate"`
	ValidFrom        time.Time  `json:"validFrom"`
	ValidUntil       *time.Time `json:"validUntil"`
	ContractType     string     `json:"contractType"`
	Status           string     `json:"status"` // active, expired, terminated
	FilePath         string     `json:"filePath"`
	Notes            string     `json:"notes"`
	CreatedBy        *string    `json:"createdBy"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type CreateContractRequest struct {
	ContractNumber string `json:"contractNumber" binding:"required"`
	OrganizationID string `json:"organizationId" binding:"required"`
	ContractDate   string `json:"contractDate" binding:"required"`
	ValidFrom      string `json:"validFrom" binding:"required"`
	ValidUntil     string `json:"validUntil"`
	ContractType   string `json:"contractType" binding:"required,oneof=standard vip temporary"`
	Notes          string `json:"notes"`
}
