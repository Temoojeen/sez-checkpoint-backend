package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type ApplicationRepository struct {
	db *sql.DB
}

func NewApplicationRepository(db *sql.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// Create - создает новую заявку
func (r *ApplicationRepository) Create(app *models.Application) error {
	query := `
        INSERT INTO applications (
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            contract_id, organization_id, list_id, applicant_id,
            status, valid_from, valid_until, notes,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
    `
	_, err := r.db.Exec(query,
		app.ID, app.PlateNumber, app.VehicleBrand, app.VehicleModel, app.VehicleColor,
		app.ContractID, app.OrganizationID, app.ListID, app.ApplicantID,
		app.Status, app.ValidFrom, app.ValidUntil, app.Notes,
		time.Now(), time.Now(),
	)
	return err
}

// GetByID - получает заявку по ID
func (r *ApplicationRepository) GetByID(id string) (*models.Application, error) {
	app := &models.Application{}
	query := `
        SELECT 
            a.id, a.plate_number, a.vehicle_brand, a.vehicle_model, a.vehicle_color,
            a.contract_id, a.organization_id, a.list_id, a.applicant_id,
            a.status, a.operator_id, a.supervisor_id,
            a.operator_approved_at, a.supervisor_approved_at,
            a.rejected_at, a.reject_reason,
            a.valid_from, a.valid_until, a.notes,
            a.created_at, a.updated_at,
            o.name as organization_name,
            al.name as list_name,
            u.full_name as applicant_name,
            c.contract_number
        FROM applications a
        LEFT JOIN organizations o ON a.organization_id = o.id
        LEFT JOIN access_lists al ON a.list_id = al.id
        LEFT JOIN users u ON a.applicant_id = u.id
        LEFT JOIN contracts c ON a.contract_id = c.id
        WHERE a.id = $1
    `

	var contractID, organizationID, operatorID, supervisorID sql.NullString
	var operatorApprovedAt, supervisorApprovedAt, rejectedAt, validFrom, validUntil sql.NullTime
	var rejectReason, notes, organizationName, listName, applicantName, contractNumber sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&app.ID, &app.PlateNumber, &app.VehicleBrand, &app.VehicleModel, &app.VehicleColor,
		&contractID, &organizationID, &app.ListID, &app.ApplicantID,
		&app.Status, &operatorID, &supervisorID,
		&operatorApprovedAt, &supervisorApprovedAt,
		&rejectedAt, &rejectReason,
		&validFrom, &validUntil, &notes,
		&app.CreatedAt, &app.UpdatedAt,
		&organizationName, &listName, &applicantName, &contractNumber,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("заявка не найдена")
	}
	if err != nil {
		log.Printf("❌ Ошибка GetByID: %v", err)
		return nil, err
	}

	// Конвертируем NULL значения
	if contractID.Valid {
		app.ContractID = &contractID.String
	}
	if organizationID.Valid {
		app.OrganizationID = &organizationID.String
	}
	if operatorID.Valid {
		app.OperatorID = &operatorID.String
	}
	if supervisorID.Valid {
		app.SupervisorID = &supervisorID.String
	}
	if operatorApprovedAt.Valid {
		app.OperatorApprovedAt = &operatorApprovedAt.Time
	}
	if supervisorApprovedAt.Valid {
		app.SupervisorApprovedAt = &supervisorApprovedAt.Time
	}
	if rejectedAt.Valid {
		app.RejectedAt = &rejectedAt.Time
	}
	if rejectReason.Valid {
		app.RejectReason = rejectReason.String
	}
	if validFrom.Valid {
		app.ValidFrom = &validFrom.Time
	}
	if validUntil.Valid {
		app.ValidUntil = &validUntil.Time
	}
	if notes.Valid {
		app.Notes = notes.String
	}
	if organizationName.Valid {
		app.OrganizationName = organizationName.String
	}
	if listName.Valid {
		app.ListName = listName.String
	}
	if applicantName.Valid {
		app.ApplicantName = applicantName.String
	}
	if contractNumber.Valid {
		app.ContractNumber = contractNumber.String
	}

	return app, nil
}

// GetByStatus - получает заявки по статусу
func (r *ApplicationRepository) GetByStatus(status string) ([]*models.Application, error) {
	rows, err := r.db.Query(`
        SELECT 
            a.id, a.plate_number, a.vehicle_brand, a.vehicle_model, a.vehicle_color,
            a.contract_id, a.organization_id, a.list_id, a.applicant_id,
            a.status, a.operator_id, a.supervisor_id,
            a.operator_approved_at, a.supervisor_approved_at,
            a.rejected_at, a.reject_reason,
            a.valid_from, a.valid_until, a.notes,
            a.created_at, a.updated_at,
            o.name as organization_name,
            al.name as list_name,
            u.full_name as applicant_name,
            c.contract_number
        FROM applications a
        LEFT JOIN organizations o ON a.organization_id = o.id
        LEFT JOIN access_lists al ON a.list_id = al.id
        LEFT JOIN users u ON a.applicant_id = u.id
        LEFT JOIN contracts c ON a.contract_id = c.id
        WHERE a.status = $1
        ORDER BY a.created_at DESC
    `, status)
	if err != nil {
		log.Printf("❌ Ошибка GetByStatus: %v", err)
		return nil, err
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		app := &models.Application{}

		var contractID, organizationID, operatorID, supervisorID sql.NullString
		var operatorApprovedAt, supervisorApprovedAt, rejectedAt, validFrom, validUntil sql.NullTime
		var rejectReason, notes, organizationName, listName, applicantName, contractNumber sql.NullString

		err := rows.Scan(
			&app.ID, &app.PlateNumber, &app.VehicleBrand, &app.VehicleModel, &app.VehicleColor,
			&contractID, &organizationID, &app.ListID, &app.ApplicantID,
			&app.Status, &operatorID, &supervisorID,
			&operatorApprovedAt, &supervisorApprovedAt,
			&rejectedAt, &rejectReason,
			&validFrom, &validUntil, &notes,
			&app.CreatedAt, &app.UpdatedAt,
			&organizationName, &listName, &applicantName, &contractNumber,
		)
		if err != nil {
			log.Printf("❌ Ошибка сканирования GetByStatus: %v", err)
			continue
		}

		// Конвертируем NULL значения
		if contractID.Valid {
			app.ContractID = &contractID.String
		}
		if organizationID.Valid {
			app.OrganizationID = &organizationID.String
		}
		if operatorID.Valid {
			app.OperatorID = &operatorID.String
		}
		if supervisorID.Valid {
			app.SupervisorID = &supervisorID.String
		}
		if operatorApprovedAt.Valid {
			app.OperatorApprovedAt = &operatorApprovedAt.Time
		}
		if supervisorApprovedAt.Valid {
			app.SupervisorApprovedAt = &supervisorApprovedAt.Time
		}
		if rejectedAt.Valid {
			app.RejectedAt = &rejectedAt.Time
		}
		if rejectReason.Valid {
			app.RejectReason = rejectReason.String
		}
		if validFrom.Valid {
			app.ValidFrom = &validFrom.Time
		}
		if validUntil.Valid {
			app.ValidUntil = &validUntil.Time
		}
		if notes.Valid {
			app.Notes = notes.String
		}
		if organizationName.Valid {
			app.OrganizationName = organizationName.String
		}
		if listName.Valid {
			app.ListName = listName.String
		}
		if applicantName.Valid {
			app.ApplicantName = applicantName.String
		}
		if contractNumber.Valid {
			app.ContractNumber = contractNumber.String
		}

		applications = append(applications, app)
	}

	if err = rows.Err(); err != nil {
		log.Printf("❌ Ошибка после итерации GetByStatus: %v", err)
		return nil, err
	}

	return applications, nil
}

// GetByApplicant - получает заявки конкретного заявителя
func (r *ApplicationRepository) GetByApplicant(applicantID string) ([]*models.Application, error) {
	rows, err := r.db.Query(`
        SELECT 
            a.id, a.plate_number, a.vehicle_brand, a.vehicle_model, a.vehicle_color,
            a.contract_id, a.organization_id, a.list_id, a.applicant_id,
            a.status, a.operator_id, a.supervisor_id,
            a.operator_approved_at, a.supervisor_approved_at,
            a.rejected_at, a.reject_reason,
            a.valid_from, a.valid_until, a.notes,
            a.created_at, a.updated_at,
            o.name as organization_name,
            al.name as list_name,
            c.contract_number
        FROM applications a
        LEFT JOIN organizations o ON a.organization_id = o.id
        LEFT JOIN access_lists al ON a.list_id = al.id
        LEFT JOIN contracts c ON a.contract_id = c.id
        WHERE a.applicant_id = $1
        ORDER BY a.created_at DESC
    `, applicantID)
	if err != nil {
		log.Printf("❌ Ошибка GetByApplicant: %v", err)
		return nil, err
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		app := &models.Application{}

		var contractID, organizationID, operatorID, supervisorID sql.NullString
		var operatorApprovedAt, supervisorApprovedAt, rejectedAt, validFrom, validUntil sql.NullTime
		var rejectReason, notes, organizationName, listName, contractNumber sql.NullString

		err := rows.Scan(
			&app.ID, &app.PlateNumber, &app.VehicleBrand, &app.VehicleModel, &app.VehicleColor,
			&contractID, &organizationID, &app.ListID, &app.ApplicantID,
			&app.Status, &operatorID, &supervisorID,
			&operatorApprovedAt, &supervisorApprovedAt,
			&rejectedAt, &rejectReason,
			&validFrom, &validUntil, &notes,
			&app.CreatedAt, &app.UpdatedAt,
			&organizationName, &listName, &contractNumber,
		)
		if err != nil {
			log.Printf("❌ Ошибка сканирования GetByApplicant: %v", err)
			continue
		}

		// Конвертируем NULL значения
		if contractID.Valid {
			app.ContractID = &contractID.String
		}
		if organizationID.Valid {
			app.OrganizationID = &organizationID.String
		}
		if operatorID.Valid {
			app.OperatorID = &operatorID.String
		}
		if supervisorID.Valid {
			app.SupervisorID = &supervisorID.String
		}
		if operatorApprovedAt.Valid {
			app.OperatorApprovedAt = &operatorApprovedAt.Time
		}
		if supervisorApprovedAt.Valid {
			app.SupervisorApprovedAt = &supervisorApprovedAt.Time
		}
		if rejectedAt.Valid {
			app.RejectedAt = &rejectedAt.Time
		}
		if rejectReason.Valid {
			app.RejectReason = rejectReason.String
		}
		if validFrom.Valid {
			app.ValidFrom = &validFrom.Time
		}
		if validUntil.Valid {
			app.ValidUntil = &validUntil.Time
		}
		if notes.Valid {
			app.Notes = notes.String
		}
		if organizationName.Valid {
			app.OrganizationName = organizationName.String
		}
		if listName.Valid {
			app.ListName = listName.String
		}
		if contractNumber.Valid {
			app.ContractNumber = contractNumber.String
		}

		applications = append(applications, app)
	}

	if err = rows.Err(); err != nil {
		log.Printf("❌ Ошибка после итерации GetByApplicant: %v", err)
		return nil, err
	}

	return applications, nil
}

// GetByOrganization - получает заявки организации
func (r *ApplicationRepository) GetByOrganization(orgID string) ([]*models.Application, error) {
	rows, err := r.db.Query(`
        SELECT 
            a.id, a.plate_number, a.vehicle_brand, a.vehicle_model, a.vehicle_color,
            a.contract_id, a.list_id, a.applicant_id,
            a.status, a.operator_id, a.supervisor_id,
            a.created_at,
            al.name as list_name,
            u.full_name as applicant_name,
            c.contract_number
        FROM applications a
        LEFT JOIN access_lists al ON a.list_id = al.id
        LEFT JOIN users u ON a.applicant_id = u.id
        LEFT JOIN contracts c ON a.contract_id = c.id
        WHERE a.organization_id = $1
        ORDER BY a.created_at DESC
    `, orgID)
	if err != nil {
		log.Printf("❌ Ошибка GetByOrganization: %v", err)
		return nil, err
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		app := &models.Application{}

		var contractID, operatorID, supervisorID sql.NullString
		var listName, applicantName, contractNumber sql.NullString

		err := rows.Scan(
			&app.ID, &app.PlateNumber, &app.VehicleBrand, &app.VehicleModel, &app.VehicleColor,
			&contractID, &app.ListID, &app.ApplicantID,
			&app.Status, &operatorID, &supervisorID,
			&app.CreatedAt,
			&listName, &applicantName, &contractNumber,
		)
		if err != nil {
			log.Printf("❌ Ошибка сканирования GetByOrganization: %v", err)
			continue
		}

		if contractID.Valid {
			app.ContractID = &contractID.String
		}
		if operatorID.Valid {
			app.OperatorID = &operatorID.String
		}
		if supervisorID.Valid {
			app.SupervisorID = &supervisorID.String
		}
		if listName.Valid {
			app.ListName = listName.String
		}
		if applicantName.Valid {
			app.ApplicantName = applicantName.String
		}
		if contractNumber.Valid {
			app.ContractNumber = contractNumber.String
		}

		applications = append(applications, app)
	}

	if err = rows.Err(); err != nil {
		log.Printf("❌ Ошибка после итерации GetByOrganization: %v", err)
		return nil, err
	}

	return applications, nil
}

// UpdateStatus - обновляет статус заявки
func (r *ApplicationRepository) UpdateStatus(
	id, status string,
	operatorID, supervisorID *string,
	operatorApprovedAt, supervisorApprovedAt *time.Time,
	rejectReason string,
) error {
	query := `
        UPDATE applications SET
            status = $1,
            operator_id = $2,
            supervisor_id = $3,
            operator_approved_at = $4,
            supervisor_approved_at = $5,
            rejected_at = $6,
            reject_reason = $7,
            updated_at = $8
        WHERE id = $9
    `

	var rejectedAt *time.Time
	if status == "rejected" {
		now := time.Now()
		rejectedAt = &now
	}

	result, err := r.db.Exec(query,
		status, operatorID, supervisorID,
		operatorApprovedAt, supervisorApprovedAt,
		rejectedAt, rejectReason, time.Now(), id,
	)
	if err != nil {
		log.Printf("❌ Ошибка UpdateStatus: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("заявка не найдена")
	}
	return nil
}

// GetPendingForOperator - получает заявки, ожидающие оператора
func (r *ApplicationRepository) GetPendingForOperator() ([]*models.Application, error) {
	return r.GetByStatus("pending")
}

// GetPendingForSupervisor - получает заявки, ожидающие руководителя
func (r *ApplicationRepository) GetPendingForSupervisor() ([]*models.Application, error) {
	return r.GetByStatus("operator_approved")
}

// GetStats - получает статистику по заявкам
func (r *ApplicationRepository) GetStats(organizationID string) (map[string]int, error) {
	query := `
        SELECT 
            COUNT(*) as total,
            COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
            COUNT(CASE WHEN status = 'operator_approved' THEN 1 END) as operator_approved,
            COUNT(CASE WHEN status = 'supervisor_approved' THEN 1 END) as approved,
            COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected
        FROM applications
    `

	var args []interface{}
	if organizationID != "" {
		query += " WHERE organization_id = $1"
		args = append(args, organizationID)
	}

	var total, pending, operatorApproved, approved, rejected int
	err := r.db.QueryRow(query, args...).Scan(
		&total, &pending, &operatorApproved, &approved, &rejected,
	)
	if err != nil {
		log.Printf("❌ Ошибка GetStats: %v", err)
		return nil, err
	}

	stats := map[string]int{
		"total":            total,
		"pending":          pending,
		"operatorApproved": operatorApproved,
		"approved":         approved,
		"rejected":         rejected,
	}
	return stats, nil
}

// AddToApprovedPlates - добавляет номер в утвержденные после одобрения
func (r *ApplicationRepository) AddToApprovedPlates(plate *models.ApprovedPlate) error {
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
		log.Printf("❌ Ошибка в AddToApprovedPlates: %v", err)
	}
	return err
}

// GetAllFiltered - получает все заявки с фильтрацией (для админа)
func (r *ApplicationRepository) GetAllFiltered(status, organizationID, listID, fromDate, toDate string) ([]*models.Application, error) {
	query := `
        SELECT 
            a.id, a.plate_number, a.vehicle_brand, a.vehicle_model, a.vehicle_color,
            a.contract_id, a.organization_id, a.list_id, a.applicant_id,
            a.status, a.operator_id, a.supervisor_id,
            a.operator_approved_at, a.supervisor_approved_at,
            a.rejected_at, a.reject_reason,
            a.valid_from, a.valid_until, a.notes,
            a.created_at, a.updated_at,
            o.name as organization_name,
            al.name as list_name,
            u.full_name as applicant_name,
            c.contract_number
        FROM applications a
        LEFT JOIN organizations o ON a.organization_id = o.id
        LEFT JOIN access_lists al ON a.list_id = al.id
        LEFT JOIN users u ON a.applicant_id = u.id
        LEFT JOIN contracts c ON a.contract_id = c.id
        WHERE 1=1
    `
	var args []interface{}
	argCount := 1

	if status != "" {
		query += " AND a.status = $" + fmt.Sprint(argCount)
		args = append(args, status)
		argCount++
	}

	if organizationID != "" {
		query += " AND a.organization_id = $" + fmt.Sprint(argCount)
		args = append(args, organizationID)
		argCount++
	}

	if listID != "" {
		query += " AND a.list_id = $" + fmt.Sprint(argCount)
		args = append(args, listID)
		argCount++
	}

	if fromDate != "" {
		query += " AND a.created_at >= $" + fmt.Sprint(argCount)
		args = append(args, fromDate)
		argCount++
	}

	if toDate != "" {
		query += " AND a.created_at <= $" + fmt.Sprint(argCount) + " + INTERVAL '1 day'"
		args = append(args, toDate)
		argCount++
	}

	query += " ORDER BY a.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		log.Printf("❌ Ошибка GetAllFiltered: %v", err)
		return nil, err
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		app := &models.Application{}

		var contractID, organizationID, operatorID, supervisorID sql.NullString
		var operatorApprovedAt, supervisorApprovedAt, rejectedAt, validFrom, validUntil sql.NullTime
		var rejectReason, notes, organizationName, listName, applicantName, contractNumber sql.NullString

		err := rows.Scan(
			&app.ID, &app.PlateNumber, &app.VehicleBrand, &app.VehicleModel, &app.VehicleColor,
			&contractID, &organizationID, &app.ListID, &app.ApplicantID,
			&app.Status, &operatorID, &supervisorID,
			&operatorApprovedAt, &supervisorApprovedAt,
			&rejectedAt, &rejectReason,
			&validFrom, &validUntil, &notes,
			&app.CreatedAt, &app.UpdatedAt,
			&organizationName, &listName, &applicantName, &contractNumber,
		)
		if err != nil {
			log.Printf("❌ Ошибка сканирования GetAllFiltered: %v", err)
			continue
		}

		// Конвертируем NULL значения
		if contractID.Valid {
			app.ContractID = &contractID.String
		}
		if organizationID.Valid {
			app.OrganizationID = &organizationID.String
		}
		if operatorID.Valid {
			app.OperatorID = &operatorID.String
		}
		if supervisorID.Valid {
			app.SupervisorID = &supervisorID.String
		}
		if operatorApprovedAt.Valid {
			app.OperatorApprovedAt = &operatorApprovedAt.Time
		}
		if supervisorApprovedAt.Valid {
			app.SupervisorApprovedAt = &supervisorApprovedAt.Time
		}
		if rejectedAt.Valid {
			app.RejectedAt = &rejectedAt.Time
		}
		if rejectReason.Valid {
			app.RejectReason = rejectReason.String
		}
		if validFrom.Valid {
			app.ValidFrom = &validFrom.Time
		}
		if validUntil.Valid {
			app.ValidUntil = &validUntil.Time
		}
		if notes.Valid {
			app.Notes = notes.String
		}
		if organizationName.Valid {
			app.OrganizationName = organizationName.String
		}
		if listName.Valid {
			app.ListName = listName.String
		}
		if applicantName.Valid {
			app.ApplicantName = applicantName.String
		}
		if contractNumber.Valid {
			app.ContractNumber = contractNumber.String
		}

		applications = append(applications, app)
	}

	if err = rows.Err(); err != nil {
		log.Printf("❌ Ошибка после итерации GetAllFiltered: %v", err)
		return nil, err
	}

	return applications, nil
}

// BeginTx - начинает транзакцию
func (r *ApplicationRepository) BeginTx() (*sql.Tx, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// UpdateStatusTx - обновляет статус заявки в рамках транзакции
func (r *ApplicationRepository) UpdateStatusTx(
	tx *sql.Tx,
	id, status string,
	operatorID, supervisorID *string,
	operatorApprovedAt, supervisorApprovedAt *time.Time,
	rejectReason string,
) error {
	query := `
        UPDATE applications SET
            status = $1,
            operator_id = $2,
            supervisor_id = $3,
            operator_approved_at = $4,
            supervisor_approved_at = $5,
            rejected_at = $6,
            reject_reason = $7,
            updated_at = $8
        WHERE id = $9
    `

	var rejectedAt *time.Time
	if status == "rejected" {
		now := time.Now()
		rejectedAt = &now
	}

	result, err := tx.Exec(query,
		status, operatorID, supervisorID,
		operatorApprovedAt, supervisorApprovedAt,
		rejectedAt, rejectReason, time.Now(), id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("заявка не найдена")
	}
	return nil
}

// DeleteTx - удаляет заявку в рамках транзакции
func (r *ApplicationRepository) DeleteTx(tx *sql.Tx, id string) error {
	query := `DELETE FROM applications WHERE id = $1`
	result, err := tx.Exec(query, id)
	if err != nil {
		log.Printf("❌ Ошибка при удалении заявки %s: %v", id, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("❌ Ошибка при получении количества удаленных строк: %v", err)
		return err
	}

	if rowsAffected == 0 {
		log.Printf("⚠️ Заявка %s не найдена для удаления", id)
		return errors.New("заявка не найдена")
	}

	log.Printf("✅ Заявка %s успешно удалена", id)
	return nil
}

// Delete - удаляет заявку
func (r *ApplicationRepository) Delete(id string) error {
	query := `DELETE FROM applications WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("заявка не найдена")
	}
	return nil
}
