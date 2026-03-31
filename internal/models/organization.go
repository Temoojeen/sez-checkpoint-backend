package models

import "time"

type Organization struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	BIN          string    `json:"bin"` // БИН для Казахстана
	Address      string    `json:"address"`
	ContactPhone string    `json:"contactPhone"`
	ContactEmail string    `json:"contactEmail"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateOrganizationRequest struct {
	Name         string `json:"name" binding:"required"`
	BIN          string `json:"bin" binding:"required,len=12"`
	Address      string `json:"address"`
	ContactPhone string `json:"contactPhone"`
	ContactEmail string `json:"contactEmail" binding:"omitempty,email"`
}

type UpdateOrganizationRequest struct {
	Name         string `json:"name" binding:"required"`
	BIN          string `json:"bin" binding:"required,len=12"`
	Address      string `json:"address"`
	ContactPhone string `json:"contactPhone"`
	ContactEmail string `json:"contactEmail" binding:"omitempty,email"`
}
