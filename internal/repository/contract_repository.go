package repository

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type ContractRepository struct {
	db *sql.DB
}

func NewContractRepository(db *sql.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

// Create - создает новый договор
func (r *ContractRepository) Create(contract *models.Contract) error {
	query := `
        INSERT INTO contracts (
            id, contract_number, organization_id, contract_date, 
            valid_from, valid_until, contract_type, status, 
            file_path, notes, created_by, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
    `
	_, err := r.db.Exec(query,
		contract.ID, contract.ContractNumber, contract.OrganizationID, contract.ContractDate,
		contract.ValidFrom, contract.ValidUntil, contract.ContractType, contract.Status,
		contract.FilePath, contract.Notes, contract.CreatedBy, time.Now(), time.Now(),
	)
	return err
}

// GetByID - получает договор по ID
// GetByID - получает договор по ID
func (r *ContractRepository) GetByID(id string) (*models.Contract, error) {
	contract := &models.Contract{}
	query := `
        SELECT 
            c.id, c.contract_number, c.organization_id, c.contract_date,
            c.valid_from, c.valid_until, c.contract_type, c.status,
            c.file_path, c.notes, c.created_by, c.created_at, c.updated_at,
            o.name as organization_name
        FROM contracts c
        LEFT JOIN organizations o ON c.organization_id = o.id
        WHERE c.id = $1
    `
	err := r.db.QueryRow(query, id).Scan(
		&contract.ID, &contract.ContractNumber, &contract.OrganizationID, &contract.ContractDate,
		&contract.ValidFrom, &contract.ValidUntil, &contract.ContractType, &contract.Status,
		&contract.FilePath, &contract.Notes, &contract.CreatedBy, &contract.CreatedAt, &contract.UpdatedAt,
		&contract.OrganizationName,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("договор не найден")
	}
	return contract, err
}

// GetByNumber - получает договор по номеру
// GetByNumber - получает договор по номеру
func (r *ContractRepository) GetByNumber(contractNumber string) (*models.Contract, error) {
	contract := &models.Contract{}

	var status sql.NullString
	var validUntil sql.NullTime

	query := `
        SELECT 
            id, contract_number, organization_id, valid_from, valid_until, status
        FROM contracts 
        WHERE contract_number = $1
    `
	err := r.db.QueryRow(query, contractNumber).Scan(
		&contract.ID, &contract.ContractNumber, &contract.OrganizationID,
		&contract.ValidFrom, &validUntil, &status,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("договор не найден")
	}
	if err != nil {
		return nil, err
	}

	// Обработка NULL статуса
	if status.Valid {
		contract.Status = status.String
	} else {
		contract.Status = "active" // По умолчанию активный
	}

	// Обработка NULL даты окончания
	if validUntil.Valid {
		contract.ValidUntil = &validUntil.Time
	}

	return contract, nil
}

// CheckContractExists - проверяет существует ли договор с таким номером
func (r *ContractRepository) CheckContractExists(contractNumber string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM contracts WHERE contract_number = $1 AND status = 'active')"
	err := r.db.QueryRow(query, contractNumber).Scan(&exists)
	return exists, err
}

// GetAll - получает все договоры
func (r *ContractRepository) GetAll() ([]*models.Contract, error) {
	rows, err := r.db.Query(`
        SELECT 
            c.id, c.contract_number, c.organization_id, c.contract_date,
            c.valid_from, c.valid_until, c.contract_type, c.status,
            c.file_path, c.notes, c.created_at, c.updated_at,
            o.name as organization_name
        FROM contracts c
        LEFT JOIN organizations o ON c.organization_id = o.id
        ORDER BY c.created_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []*models.Contract
	for rows.Next() {
		contract := &models.Contract{}
		err := rows.Scan(
			&contract.ID, &contract.ContractNumber, &contract.OrganizationID, &contract.ContractDate,
			&contract.ValidFrom, &contract.ValidUntil, &contract.ContractType, &contract.Status,
			&contract.FilePath, &contract.Notes, &contract.CreatedAt, &contract.UpdatedAt,
			&contract.OrganizationName,
		)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, contract)
	}
	return contracts, nil
}

// GetByOrganization - получает договоры организации
func (r *ContractRepository) GetByOrganization(orgID string) ([]*models.Contract, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, contract_number, contract_date, valid_from, valid_until,
            contract_type, status, notes, created_at
        FROM contracts
        WHERE organization_id = $1
        ORDER BY created_at DESC
    `, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []*models.Contract
	for rows.Next() {
		contract := &models.Contract{}
		err := rows.Scan(
			&contract.ID, &contract.ContractNumber, &contract.ContractDate,
			&contract.ValidFrom, &contract.ValidUntil, &contract.ContractType,
			&contract.Status, &contract.Notes, &contract.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, contract)
	}
	return contracts, nil
}

// Update - обновляет договор
func (r *ContractRepository) Update(contract *models.Contract) error {
	query := `
        UPDATE contracts SET
            contract_number = $1,
            organization_id = $2,
            contract_date = $3,
            valid_from = $4,
            valid_until = $5,
            contract_type = $6,
            status = $7,
            notes = $8,
            updated_at = $9
        WHERE id = $10
    `
	result, err := r.db.Exec(query,
		contract.ContractNumber, contract.OrganizationID, contract.ContractDate,
		contract.ValidFrom, contract.ValidUntil, contract.ContractType,
		contract.Status, contract.Notes, time.Now(), contract.ID,
	)
	if err != nil {
		log.Printf("❌ Ошибка при обновлении договора: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("договор не найден")
	}
	return nil
}

// GetByContract - получает заявки по договору (вспомогательный метод)
func (r *ApplicationRepository) GetByContract(contractID string) ([]*models.Application, error) {
	rows, err := r.db.Query(`
        SELECT id FROM applications WHERE contract_id = $1
    `, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		app := &models.Application{}
		err := rows.Scan(&app.ID)
		if err != nil {
			return nil, err
		}
		applications = append(applications, app)
	}
	return applications, nil
}

// GetByContract - получает утвержденные номера по договору
func (r *ApprovedPlateRepository) GetByContract(contractID string) ([]*models.ApprovedPlate, error) {
	rows, err := r.db.Query(`
        SELECT id FROM approved_plates WHERE contract_id = $1
    `, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plates []*models.ApprovedPlate
	for rows.Next() {
		plate := &models.ApprovedPlate{}
		err := rows.Scan(&plate.ID)
		if err != nil {
			return nil, err
		}
		plates = append(plates, plate)
	}
	return plates, nil
}

// Delete - удаляет договор
func (r *ContractRepository) Delete(id string) error {
	query := `DELETE FROM contracts WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("договор не найден")
	}
	return nil
}
