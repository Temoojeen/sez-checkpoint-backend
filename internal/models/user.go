package models

import "time"

type User struct {
	ID               string     `json:"id"`
	Username         string     `json:"username"`
	PasswordHash     string     `json:"-"`
	FullName         string     `json:"fullName"`
	Email            *string    `json:"email"` // Изменено на указатель
	Phone            *string    `json:"phone"` // Изменено на указатель
	OrganizationID   *string    `json:"organizationId"`
	OrganizationName string     `json:"organizationName,omitempty"`
	RoleID           int        `json:"roleId"`
	RoleName         string     `json:"roleName,omitempty"`
	IsActive         bool       `json:"isActive"`
	CreatedBy        *string    `json:"createdBy"`
	LastLogin        *time.Time `json:"lastLogin"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type CreateUserRequest struct {
	Username       string  `json:"username" binding:"required,min=3,max=100"`
	Password       string  `json:"password" binding:"required,min=6"`
	FullName       string  `json:"fullName" binding:"required"`
	Email          *string `json:"email" binding:"omitempty,email"` // Изменено на указатель
	Phone          *string `json:"phone"`                           // Изменено на указатель
	OrganizationID *string `json:"organizationId"`
	RoleID         int     `json:"roleId" binding:"required,min=1,max=5"`
}

type UpdateUserRequest struct {
	FullName       string  `json:"fullName" binding:"required"`
	Email          *string `json:"email" binding:"omitempty,email"` // Изменено на указатель
	Phone          *string `json:"phone"`                           // Изменено на указатель
	OrganizationID *string `json:"organizationId"`
	RoleID         int     `json:"roleId" binding:"required,min=1,max=5"`
	IsActive       *bool   `json:"isActive"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}
