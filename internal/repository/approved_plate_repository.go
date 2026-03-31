package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type ApprovedPlateRepository struct {
	db *sql.DB
}

func NewApprovedPlateRepository(db *sql.DB) *ApprovedPlateRepository {
	return &ApprovedPlateRepository{db: db}
}

// BeginTx - начинает транзакцию
func (r *ApprovedPlateRepository) BeginTx() (*sql.Tx, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// Create - добавляет утвержденный номер
func (r *ApprovedPlateRepository) Create(plate *models.ApprovedPlate) error {
	query := `
        INSERT INTO approved_plates (
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            contract_id, organization_id, list_id, application_id,
            approved_by, valid_from, valid_until, is_active, notes,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
    `
	_, err := r.db.Exec(query,
		plate.ID, plate.PlateNumber, plate.VehicleBrand, plate.VehicleModel, plate.VehicleColor,
		plate.ContractID, plate.OrganizationID, plate.ListID, plate.ApplicationID,
		plate.ApprovedBy, plate.ValidFrom, plate.ValidUntil, plate.IsActive, plate.Notes,
		time.Now(), time.Now(),
	)
	if err != nil {
		log.Printf("❌ Ошибка при создании записи в approved_plates: %v", err)
		return err
	}
	return nil
}

// CreateTx - добавляет утвержденный номер в рамках транзакции
func (r *ApprovedPlateRepository) CreateTx(tx *sql.Tx, plate *models.ApprovedPlate) error {
	query := `
        INSERT INTO approved_plates (
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            contract_id, organization_id, list_id, application_id,
            approved_by, valid_from, valid_until, is_active, notes,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
    `
	_, err := tx.Exec(query,
		plate.ID, plate.PlateNumber, plate.VehicleBrand, plate.VehicleModel, plate.VehicleColor,
		plate.ContractID, plate.OrganizationID, plate.ListID, plate.ApplicationID,
		plate.ApprovedBy, plate.ValidFrom, plate.ValidUntil, plate.IsActive, plate.Notes,
		time.Now(), time.Now(),
	)
	return err
}

// GetByPlateNumber - ищет номер в утвержденном списке
func (r *ApprovedPlateRepository) GetByPlateNumber(plateNumber string) (*models.ApprovedPlate, error) {
	plate := &models.ApprovedPlate{}
	query := `
        SELECT 
            ap.id, ap.plate_number, ap.vehicle_brand, ap.vehicle_model, ap.vehicle_color,
            ap.contract_id, ap.organization_id, ap.list_id, ap.application_id,
            ap.approved_by, ap.valid_from, ap.valid_until, ap.is_active, ap.notes,
            ap.created_at, ap.updated_at,
            o.name as organization_name,
            al.name as list_name,
            al.list_type,
            al.color as list_color  -- Добавляем color
        FROM approved_plates ap
        LEFT JOIN organizations o ON ap.organization_id = o.id
        LEFT JOIN access_lists al ON ap.list_id = al.id
        WHERE ap.plate_number = $1 
          AND ap.is_active = true 
          AND (ap.valid_until IS NULL OR ap.valid_until >= CURRENT_DATE)
        LIMIT 1
    `
	err := r.db.QueryRow(query, plateNumber).Scan(
		&plate.ID, &plate.PlateNumber, &plate.VehicleBrand, &plate.VehicleModel, &plate.VehicleColor,
		&plate.ContractID, &plate.OrganizationID, &plate.ListID, &plate.ApplicationID,
		&plate.ApprovedBy, &plate.ValidFrom, &plate.ValidUntil, &plate.IsActive, &plate.Notes,
		&plate.CreatedAt, &plate.UpdatedAt,
		&plate.OrganizationName, &plate.ListName, &plate.ListType, &plate.ListColor, // Добавляем ListColor
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("номер не найден в списке пропусков")
	}
	return plate, err
}

// GetAll - получает все утвержденные номера с фильтрацией
func (r *ApprovedPlateRepository) GetAll(organizationID, listID string, onlyActive bool) ([]*models.ApprovedPlate, error) {
	query := `
        SELECT 
            ap.id, ap.plate_number, ap.vehicle_brand, ap.vehicle_model, ap.vehicle_color,
            ap.contract_id, ap.organization_id, ap.list_id, ap.application_id,
            ap.approved_by, ap.valid_from, ap.valid_until, ap.is_active, ap.notes,
            ap.created_at, ap.updated_at,
            o.name as organization_name,
            al.name as list_name,
            al.list_type,
            al.color as list_color,
            u.full_name as approved_by_name
        FROM approved_plates ap
        LEFT JOIN organizations o ON ap.organization_id = o.id
        LEFT JOIN access_lists al ON ap.list_id = al.id
        LEFT JOIN users u ON ap.approved_by = u.id
        WHERE 1=1
    `

	var args []interface{}
	argCount := 1

	if organizationID != "" {
		query += " AND ap.organization_id = $" + fmt.Sprint(argCount)
		args = append(args, organizationID)
		argCount++
	}

	if listID != "" {
		query += " AND ap.list_id = $" + fmt.Sprint(argCount)
		args = append(args, listID)
		argCount++
	}

	// Фильтруем только активные, если параметр onlyActive = true
	// По умолчанию возвращаем все номера (включая неактивные)
	if onlyActive {
		query += " AND ap.is_active = true AND (ap.valid_until IS NULL OR ap.valid_until >= CURRENT_DATE)"
	}

	query += " ORDER BY ap.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plates []*models.ApprovedPlate
	for rows.Next() {
		plate := &models.ApprovedPlate{}
		var contractID, organizationID, applicationID, approvedBy sql.NullString
		var validFrom, validUntil sql.NullTime
		var notes sql.NullString
		var organizationName, listName, listType, listColor, approvedByName sql.NullString

		err := rows.Scan(
			&plate.ID, &plate.PlateNumber, &plate.VehicleBrand, &plate.VehicleModel, &plate.VehicleColor,
			&contractID, &organizationID, &plate.ListID, &applicationID,
			&approvedBy, &validFrom, &validUntil, &plate.IsActive, &notes,
			&plate.CreatedAt, &plate.UpdatedAt,
			&organizationName, &listName, &listType, &listColor, &approvedByName,
		)
		if err != nil {
			return nil, err
		}

		// Конвертируем NULL значения
		if contractID.Valid {
			plate.ContractID = &contractID.String
		}
		if organizationID.Valid {
			plate.OrganizationID = &organizationID.String
		}
		if applicationID.Valid {
			plate.ApplicationID = &applicationID.String
		}
		if approvedBy.Valid {
			plate.ApprovedBy = &approvedBy.String
		}
		if validFrom.Valid {
			plate.ValidFrom = &validFrom.Time
		}
		if validUntil.Valid {
			plate.ValidUntil = &validUntil.Time
		}
		if notes.Valid {
			plate.Notes = notes.String
		}
		if organizationName.Valid {
			plate.OrganizationName = organizationName.String
		}
		if listName.Valid {
			plate.ListName = listName.String
		}
		if listType.Valid {
			plate.ListType = listType.String
		}
		if listColor.Valid {
			plate.ListColor = listColor.String
		}
		if approvedByName.Valid {
			plate.ApprovedByName = approvedByName.String
		}

		plates = append(plates, plate)
	}
	return plates, nil
}

// GetByOrganization - получает все номера организации
func (r *ApprovedPlateRepository) GetByOrganization(organizationID string) ([]*models.ApprovedPlate, error) {
	return r.GetAll(organizationID, "", true)
}

// GetByList - получает все номера в конкретном списке
func (r *ApprovedPlateRepository) GetByList(listID string) ([]*models.ApprovedPlate, error) {
	return r.GetAll("", listID, true)
}

// GetByPlateNumberAndList - получает номер по его значению и ID списка
func (r *ApprovedPlateRepository) GetByPlateNumberAndList(plateNumber, listID string) (*models.ApprovedPlate, error) {
	plate := &models.ApprovedPlate{}
	query := `
        SELECT 
            id, plate_number, created_at
        FROM approved_plates 
        WHERE plate_number = $1 AND list_id = $2 AND is_active = true
        LIMIT 1
    `
	err := r.db.QueryRow(query, plateNumber, listID).Scan(
		&plate.ID, &plate.PlateNumber, &plate.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("номер не найден")
	}
	return plate, err
}

// GetByPlateNumberAndListIncludeInactive - получает номер по его значению и ID списка (включая неактивные)
func (r *ApprovedPlateRepository) GetByPlateNumberAndListIncludeInactive(plateNumber, listID string) (*models.ApprovedPlate, error) {
	plate := &models.ApprovedPlate{}
	query := `
        SELECT 
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            organization_id, list_id, valid_from, valid_until, is_active, notes,
            created_at, updated_at
        FROM approved_plates 
        WHERE plate_number = $1 AND list_id = $2
        LIMIT 1
    `
	err := r.db.QueryRow(query, plateNumber, listID).Scan(
		&plate.ID, &plate.PlateNumber, &plate.VehicleBrand, &plate.VehicleModel, &plate.VehicleColor,
		&plate.OrganizationID, &plate.ListID, &plate.ValidFrom, &plate.ValidUntil,
		&plate.IsActive, &plate.Notes, &plate.CreatedAt, &plate.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("номер не найден")
	}
	return plate, err
}

// Update - обновляет данные утвержденного номера
func (r *ApprovedPlateRepository) Update(plate *models.ApprovedPlate) error {
	query := `
        UPDATE approved_plates SET
            plate_number = $1,
            vehicle_brand = $2,
            vehicle_model = $3,
            vehicle_color = $4,
            list_id = $5,
            valid_from = $6,
            valid_until = $7,
            is_active = $8,
            notes = $9,
            updated_at = $10
        WHERE id = $11
    `
	result, err := r.db.Exec(query,
		plate.PlateNumber, plate.VehicleBrand, plate.VehicleModel, plate.VehicleColor,
		plate.ListID, plate.ValidFrom, plate.ValidUntil, plate.IsActive, plate.Notes,
		time.Now(), plate.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("запись не найдена")
	}
	return nil
}

// Delete - мягкое удаление (деактивация)
func (r *ApprovedPlateRepository) Delete(id string) error {
	query := `UPDATE approved_plates SET is_active = false, updated_at = $1 WHERE id = $2`
	result, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("запись не найдена")
	}
	return nil
}

// HardDelete - полное удаление записи из базы
func (r *ApprovedPlateRepository) HardDelete(id string) error {
	query := `DELETE FROM approved_plates WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("запись не найдена")
	}
	return nil
}

// CheckIfExists - проверяет существует ли номер в списке
func (r *ApprovedPlateRepository) CheckIfExists(plateNumber, listID string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM approved_plates 
            WHERE plate_number = $1 AND list_id = $2 AND is_active = true
        )
    `
	err := r.db.QueryRow(query, plateNumber, listID).Scan(&exists)
	return exists, err
}

// CheckIfExistsTx - проверяет существует ли номер в списке (в рамках транзакции)
func (r *ApprovedPlateRepository) CheckIfExistsTx(tx *sql.Tx, plateNumber, listID string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM approved_plates 
            WHERE plate_number = $1 AND list_id = $2 AND is_active = true
        )
    `
	err := tx.QueryRow(query, plateNumber, listID).Scan(&exists)
	return exists, err
}

// ReactivateByPlateAndListTx - реактивирует номер по номеру и списку
func (r *ApprovedPlateRepository) ReactivateByPlateAndListTx(tx *sql.Tx, plateNumber, listID string) error {
	query := `
        UPDATE approved_plates 
        SET is_active = true, updated_at = $1
        WHERE plate_number = $2 AND list_id = $3
    `
	_, err := tx.Exec(query, time.Now(), plateNumber, listID)
	return err
}

// GetExpiringSoon - получает номера с истекающим сроком
func (r *ApprovedPlateRepository) GetExpiringSoon(days int) ([]*models.ApprovedPlate, error) {
	rows, err := r.db.Query(`
        SELECT 
            ap.id, ap.plate_number, ap.organization_id, ap.list_id,
            ap.valid_until,
            o.name as organization_name,
            al.name as list_name
        FROM approved_plates ap
        LEFT JOIN organizations o ON ap.organization_id = o.id
        LEFT JOIN access_lists al ON ap.list_id = al.id
        WHERE ap.is_active = true 
          AND ap.valid_until IS NOT NULL
          AND ap.valid_until BETWEEN CURRENT_DATE AND CURRENT_DATE + $1
        ORDER BY ap.valid_until
    `, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plates []*models.ApprovedPlate
	for rows.Next() {
		plate := &models.ApprovedPlate{}
		err := rows.Scan(
			&plate.ID, &plate.PlateNumber, &plate.OrganizationID, &plate.ListID,
			&plate.ValidUntil, &plate.OrganizationName, &plate.ListName,
		)
		if err != nil {
			return nil, err
		}
		plates = append(plates, plate)
	}
	return plates, nil
}

// GetByID - получает номер по ID
func (r *ApprovedPlateRepository) GetByID(id string) (*models.ApprovedPlate, error) {
	plate := &models.ApprovedPlate{}
	query := `
        SELECT 
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            contract_id, organization_id, list_id, application_id,
            approved_by, valid_from, valid_until, is_active, notes,
            created_at, updated_at
        FROM approved_plates 
        WHERE id = $1
    `
	err := r.db.QueryRow(query, id).Scan(
		&plate.ID, &plate.PlateNumber, &plate.VehicleBrand, &plate.VehicleModel, &plate.VehicleColor,
		&plate.ContractID, &plate.OrganizationID, &plate.ListID, &plate.ApplicationID,
		&plate.ApprovedBy, &plate.ValidFrom, &plate.ValidUntil, &plate.IsActive, &plate.Notes,
		&plate.CreatedAt, &plate.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("номер не найден")
	}
	return plate, err
}

// GetByPlateNumberIncludeInactive - ищет номер в утвержденном списке (включая неактивные)
func (r *ApprovedPlateRepository) GetByPlateNumberIncludeInactive(plateNumber string) (*models.ApprovedPlate, error) {
	plate := &models.ApprovedPlate{}
	query := `
        SELECT 
            ap.id, ap.plate_number, ap.vehicle_brand, ap.vehicle_model, ap.vehicle_color,
            ap.contract_id, ap.organization_id, ap.list_id, ap.application_id,
            ap.approved_by, ap.valid_from, ap.valid_until, ap.is_active, ap.notes,
            ap.created_at, ap.updated_at,
            o.name as organization_name,
            al.name as list_name,
            al.list_type,
            al.color as list_color
        FROM approved_plates ap
        LEFT JOIN organizations o ON ap.organization_id = o.id
        LEFT JOIN access_lists al ON ap.list_id = al.id
        WHERE ap.plate_number = $1
        LIMIT 1
    `
	err := r.db.QueryRow(query, plateNumber).Scan(
		&plate.ID, &plate.PlateNumber, &plate.VehicleBrand, &plate.VehicleModel, &plate.VehicleColor,
		&plate.ContractID, &plate.OrganizationID, &plate.ListID, &plate.ApplicationID,
		&plate.ApprovedBy, &plate.ValidFrom, &plate.ValidUntil, &plate.IsActive, &plate.Notes,
		&plate.CreatedAt, &plate.UpdatedAt,
		&plate.OrganizationName, &plate.ListName, &plate.ListType, &plate.ListColor,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("номер не найден в списке пропусков")
	}
	return plate, err
}
