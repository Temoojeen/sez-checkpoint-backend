package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type OrganizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create - создает новую организацию
func (r *OrganizationRepository) Create(org *models.Organization) error {
	query := `
        INSERT INTO organizations (
            id, name, bin, address, contact_phone, contact_email, 
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
	_, err := r.db.Exec(query,
		org.ID, org.Name, org.BIN, org.Address, org.ContactPhone, org.ContactEmail,
		time.Now(), time.Now(),
	)
	return err
}

// GetByID - получает организацию по ID
func (r *OrganizationRepository) GetByID(id string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
        SELECT 
            id, name, bin, address, contact_phone, contact_email,
            created_at, updated_at
        FROM organizations 
        WHERE id = $1
    `
	err := r.db.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.BIN, &org.Address, &org.ContactPhone, &org.ContactEmail,
		&org.CreatedAt, &org.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("организация не найдена")
	}
	return org, err
}

// GetByBIN - получает организацию по БИН
func (r *OrganizationRepository) GetByBIN(bin string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
        SELECT 
            id, name, bin, address, contact_phone, contact_email,
            created_at, updated_at
        FROM organizations 
        WHERE bin = $1
    `
	err := r.db.QueryRow(query, bin).Scan(
		&org.ID, &org.Name, &org.BIN, &org.Address, &org.ContactPhone, &org.ContactEmail,
		&org.CreatedAt, &org.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("организация не найдена")
	}
	return org, err
}

// GetAll - получает все организации
func (r *OrganizationRepository) GetAll() ([]*models.Organization, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, name, bin, address, contact_phone, contact_email,
            created_at, updated_at
        FROM organizations
        ORDER BY name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var organizations []*models.Organization
	for rows.Next() {
		org := &models.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.BIN, &org.Address, &org.ContactPhone, &org.ContactEmail,
			&org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		organizations = append(organizations, org)
	}
	return organizations, nil
}

// Update - обновляет данные организации
func (r *OrganizationRepository) Update(org *models.Organization) error {
	query := `
        UPDATE organizations SET
            name = $1,
            bin = $2,
            address = $3,
            contact_phone = $4,
            contact_email = $5,
            updated_at = $6
        WHERE id = $7
    `
	result, err := r.db.Exec(query,
		org.Name, org.BIN, org.Address, org.ContactPhone, org.ContactEmail,
		time.Now(), org.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("организация не найдена")
	}
	return nil
}

// Delete - удаляет организацию и все связанные данные
func (r *OrganizationRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Удаляем все номера организации из approved_plates
	_, err = tx.Exec(`DELETE FROM approved_plates WHERE organization_id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления номеров организации: %v", err)
	}

	// 2. Отвязываем deleted_plates от организации
	_, err = tx.Exec(`UPDATE deleted_plates SET organization_id = NULL WHERE organization_id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления deleted_plates: %v", err)
	}

	// 3. Удаляем все заявки организации
	_, err = tx.Exec(`DELETE FROM applications WHERE organization_id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления заявок организации: %v", err)
	}

	// 4. Удаляем договоры организации
	_, err = tx.Exec(`DELETE FROM contracts WHERE organization_id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления договоров организации: %v", err)
	}

	// 5. Удаляем пользователей организации
	_, err = tx.Exec(`DELETE FROM users WHERE organization_id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователей организации: %v", err)
	}

	// 6. Удаляем саму организацию
	result, err := tx.Exec(`DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления организации: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("организация не найдена")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %v", err)
	}

	return nil
}

// GetWithStats - получает организацию со статистикой
func (r *OrganizationRepository) GetWithStats(id string) (map[string]interface{}, error) {
	var (
		orgName, bin, address, contactPhone, contactEmail string
		createdAt, updatedAt                              time.Time
		usersCount, contractsCount, platesCount           int
	)

	query := `
        SELECT 
            o.name, o.bin, o.address, o.contact_phone, o.contact_email,
            o.created_at, o.updated_at,
            COUNT(DISTINCT u.id) as users_count,
            COUNT(DISTINCT c.id) as contracts_count,
            COUNT(DISTINCT ap.id) as plates_count
        FROM organizations o
        LEFT JOIN users u ON o.id = u.organization_id AND u.is_active = true
        LEFT JOIN contracts c ON o.id = c.organization_id AND c.status = 'active'
        LEFT JOIN approved_plates ap ON o.id = ap.organization_id AND ap.is_active = true
        WHERE o.id = $1
        GROUP BY o.id
    `

	err := r.db.QueryRow(query, id).Scan(
		&orgName, &bin, &address, &contactPhone, &contactEmail,
		&createdAt, &updatedAt,
		&usersCount, &contractsCount, &platesCount,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("организация не найдена")
	}
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":             id,
		"name":           orgName,
		"bin":            bin,
		"address":        address,
		"contactPhone":   contactPhone,
		"contactEmail":   contactEmail,
		"createdAt":      createdAt,
		"updatedAt":      updatedAt,
		"usersCount":     usersCount,
		"contractsCount": contractsCount,
		"platesCount":    platesCount,
	}

	return result, nil
}

// CheckBINExists - проверяет существует ли БИН
func (r *OrganizationRepository) CheckBINExists(bin string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE bin = $1)`
	err := r.db.QueryRow(query, bin).Scan(&exists)
	return exists, err
}

// Search - поиск организаций по названию или БИН
func (r *OrganizationRepository) Search(query string) ([]*models.Organization, error) {
	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(`
        SELECT 
            id, name, bin, address, contact_phone, contact_email,
            created_at, updated_at
        FROM organizations
        WHERE name ILIKE $1 OR bin ILIKE $1
        ORDER BY name
        LIMIT 20
    `, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var organizations []*models.Organization
	for rows.Next() {
		org := &models.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.BIN, &org.Address, &org.ContactPhone, &org.ContactEmail,
			&org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		organizations = append(organizations, org)
	}
	return organizations, nil
}
