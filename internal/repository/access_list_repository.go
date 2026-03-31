package repository

import (
	"database/sql"
	"errors"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type AccessListRepository struct {
	db *sql.DB
}

func NewAccessListRepository(db *sql.DB) *AccessListRepository {
	return &AccessListRepository{db: db}
}

// Create - создает новый список доступа
func (r *AccessListRepository) Create(list *models.AccessList) error {
	query := `
        INSERT INTO access_lists (
            id, name, description, color, priority, 
            is_active, created_by, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
	_, err := r.db.Exec(query,
		list.ID, list.Name, list.Description, list.Color,
		list.Priority, list.IsActive, list.CreatedBy, time.Now(), time.Now(),
	)
	return err
}

// GetByID - получает список по ID
func (r *AccessListRepository) GetByID(id string) (*models.AccessList, error) {
	list := &models.AccessList{}
	query := `
        SELECT 
            id, name, description, color, priority, 
            is_active, created_by, created_at, updated_at
        FROM access_lists 
        WHERE id = $1
    `
	err := r.db.QueryRow(query, id).Scan(
		&list.ID, &list.Name, &list.Description, &list.Color,
		&list.Priority, &list.IsActive, &list.CreatedBy, &list.CreatedAt, &list.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("список не найден")
	}
	return list, err
}

// GetAll - получает все списки доступа
func (r *AccessListRepository) GetAll(onlyActive bool) ([]*models.AccessList, error) {
	query := `
        SELECT 
            id, name, description, color, priority, 
            is_active, created_by, created_at, updated_at
        FROM access_lists
        WHERE 1=1
    `

	if onlyActive {
		query += " AND is_active = true"
	}

	query += " ORDER BY priority, name"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []*models.AccessList
	for rows.Next() {
		list := &models.AccessList{}
		err := rows.Scan(
			&list.ID, &list.Name, &list.Description, &list.Color,
			&list.Priority, &list.IsActive, &list.CreatedBy, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, nil
}

// Update - обновляет список доступа
func (r *AccessListRepository) Update(list *models.AccessList) error {
	query := `
        UPDATE access_lists SET
            name = $1,
            description = $2,
            color = $3,
            priority = $4,
            is_active = $5,
            updated_at = $6
        WHERE id = $7
    `
	result, err := r.db.Exec(query,
		list.Name, list.Description, list.Color,
		list.Priority, list.IsActive, time.Now(), list.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("список не найден")
	}
	return nil
}

// Delete - мягкое удаление (деактивация) списка
func (r *AccessListRepository) Delete(id string) error {
	query := `UPDATE access_lists SET is_active = false, updated_at = $1 WHERE id = $2`
	result, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("список не найден")
	}
	return nil
}

// GetListsWithPermissions - получает списки с информацией о количестве пользователей
func (r *AccessListRepository) GetListsWithPermissions() ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
        SELECT 
            al.id, al.name, al.description, al.color, 
            al.priority, al.is_active, al.created_at,
            COUNT(ulp.user_id) as users_count
        FROM access_lists al
        LEFT JOIN user_list_permissions ulp ON al.id = ulp.list_id
        GROUP BY al.id
        ORDER BY al.priority, al.name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []map[string]interface{}
	for rows.Next() {
		var (
			id, name, description, color string
			priority, usersCount         int
			isActive                     bool
			createdAt                    time.Time
		)
		err := rows.Scan(
			&id, &name, &description, &color,
			&priority, &isActive, &createdAt, &usersCount,
		)
		if err != nil {
			return nil, err
		}

		list := map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"color":       color,
			"priority":    priority,
			"isActive":    isActive,
			"createdAt":   createdAt,
			"usersCount":  usersCount,
		}
		lists = append(lists, list)
	}
	return lists, nil
}
