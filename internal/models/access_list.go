// Это для информации, на бэкенде в Go должно быть так:

package models

import "time"

type AccessList struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	Priority    int       `json:"priority"`
	IsActive    bool      `json:"isActive"`
	CreatedBy   *string   `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateAccessListRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Priority    int    `json:"priority"`
}
