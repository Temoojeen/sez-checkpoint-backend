package models

import "time"

type AccessLog struct {
	ID               string    `json:"id"`
	PlateNumber      string    `json:"plateNumber"`
	OrganizationName string    `json:"organizationName"`
	ListName         string    `json:"listName"`
	ImagePath        string    `json:"imagePath"`
	AccessGranted    bool      `json:"accessGranted"`
	CameraID         string    `json:"cameraId,omitempty"`
	CameraLocation   string    `json:"cameraLocation,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type CreateAccessLogRequest struct {
	PlateNumber      string `json:"plateNumber" binding:"required"`
	OrganizationName string `json:"organizationName"`
	ListName         string `json:"listName"`
	ImagePath        string `json:"imagePath"`
	CameraID         string `json:"cameraId"`
	CameraLocation   string `json:"cameraLocation"`
}

type CheckPlateResponse struct {
	Exists           bool   `json:"exists"`
	PlateNumber      string `json:"plateNumber,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
	ListName         string `json:"listName,omitempty"`
	ListType         string `json:"listType,omitempty"`
	ListColor        string `json:"listColor,omitempty"` // Добавляем
	ValidUntil       string `json:"validUntil,omitempty"`
	IsActive         bool   `json:"isActive,omitempty"`
	Message          string `json:"message"`
}

type AccessLogStats struct {
	Date    string `json:"date"`
	Total   int    `json:"total"`
	Granted int    `json:"granted"`
	Denied  int    `json:"denied"`
}
