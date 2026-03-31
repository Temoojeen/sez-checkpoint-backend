package repository

import (
	"database/sql"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type AccessLogRepository struct {
	db *sql.DB
}

func NewAccessLogRepository(db *sql.DB) *AccessLogRepository {
	return &AccessLogRepository{db: db}
}

// Create - создает запись о проезде
func (r *AccessLogRepository) Create(log *models.AccessLog) error {
	query := `
        INSERT INTO access_logs (
            id, plate_number, organization_name, list_name, 
            image_path, access_granted, camera_id, camera_location, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
	_, err := r.db.Exec(query,
		log.ID, log.PlateNumber, log.OrganizationName, log.ListName,
		log.ImagePath, log.AccessGranted, log.CameraID, log.CameraLocation,
		log.CreatedAt,
	)
	return err
}

// GetRecent - получает последние N записей
func (r *AccessLogRepository) GetRecent(limit int) ([]*models.AccessLog, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, plate_number, organization_name, list_name, 
            image_path, access_granted, created_at
        FROM access_logs
        ORDER BY created_at DESC
        LIMIT $1
    `, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AccessLog
	for rows.Next() {
		log := &models.AccessLog{}
		err := rows.Scan(
			&log.ID, &log.PlateNumber, &log.OrganizationName, &log.ListName,
			&log.ImagePath, &log.AccessGranted, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetByDateRange - получает записи за период
func (r *AccessLogRepository) GetByDateRange(from, to time.Time) ([]*models.AccessLog, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, plate_number, organization_name, list_name, 
            image_path, access_granted, created_at
        FROM access_logs
        WHERE created_at BETWEEN $1 AND $2
        ORDER BY created_at DESC
    `, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AccessLog
	for rows.Next() {
		log := &models.AccessLog{}
		err := rows.Scan(
			&log.ID, &log.PlateNumber, &log.OrganizationName, &log.ListName,
			&log.ImagePath, &log.AccessGranted, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetByPlateNumber - получает историю проездов по номеру
func (r *AccessLogRepository) GetByPlateNumber(plateNumber string) ([]*models.AccessLog, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, plate_number, organization_name, list_name, 
            image_path, access_granted, created_at
        FROM access_logs
        WHERE plate_number = $1
        ORDER BY created_at DESC
    `, plateNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AccessLog
	for rows.Next() {
		log := &models.AccessLog{}
		err := rows.Scan(
			&log.ID, &log.PlateNumber, &log.OrganizationName, &log.ListName,
			&log.ImagePath, &log.AccessGranted, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetTodayStats - получает статистику за сегодня
func (r *AccessLogRepository) GetTodayStats() (map[string]interface{}, error) {
	today := time.Now().Format("2006-01-02")
	query := `
        SELECT 
            COUNT(*) as total,
            COUNT(CASE WHEN access_granted = true THEN 1 END) as granted,
            COUNT(CASE WHEN access_granted = false THEN 1 END) as denied
        FROM access_logs
        WHERE DATE(created_at) = $1
    `

	var total, granted, denied int
	err := r.db.QueryRow(query, today).Scan(&total, &granted, &denied)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"date":    today,
		"total":   total,
		"granted": granted,
		"denied":  denied,
	}
	return stats, nil
}
