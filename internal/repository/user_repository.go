package repository

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create - создает нового пользователя
func (r *UserRepository) Create(user *models.User) error {
	query := `
        INSERT INTO users (
            id, username, password_hash, full_name, email, phone, 
            organization_id, role_id, is_active, created_by, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `
	_, err := r.db.Exec(query,
		user.ID, user.Username, user.PasswordHash, user.FullName,
		user.Email, user.Phone, user.OrganizationID, user.RoleID,
		user.IsActive, user.CreatedBy, time.Now(), time.Now(),
	)
	return err
}

// GetByUsername - получает пользователя по имени пользователя
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT 
            id, username, password_hash, full_name, email, phone, 
            organization_id, role_id, is_active, created_by, last_login, 
            created_at, updated_at
        FROM users 
        WHERE username = $1 AND is_active = true
    `

	var email, phone sql.NullString
	var organizationID, createdBy sql.NullString
	var lastLogin sql.NullTime

	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.FullName,
		&email, &phone,
		&organizationID, &user.RoleID,
		&user.IsActive, &createdBy, &lastLogin,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("пользователь не найден")
	}
	if err != nil {
		log.Printf("❌ Ошибка при поиске пользователя: %v", err)
		return nil, err
	}

	// Конвертируем NULL значения в указатели
	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}
	if organizationID.Valid {
		user.OrganizationID = &organizationID.String
	}
	if createdBy.Valid { // Используем createdBy
		user.CreatedBy = &createdBy.String
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return user, nil
}

// GetByID - получает пользователя по ID
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT 
            u.id, u.username, u.password_hash, u.full_name, u.email, u.phone,
            u.organization_id, u.role_id, u.is_active, u.created_by, u.last_login,
            u.created_at, u.updated_at,
            o.name as organization_name,
            r.name as role_name
        FROM users u
        LEFT JOIN organizations o ON u.organization_id = o.id
        LEFT JOIN roles r ON u.role_id = r.id
        WHERE u.id = $1
    `

	var email, phone, organizationName, roleName sql.NullString
	var organizationID, createdBy sql.NullString
	var lastLogin sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.FullName,
		&email, &phone,
		&organizationID, &user.RoleID,
		&user.IsActive, &createdBy, &lastLogin,
		&user.CreatedAt, &user.UpdatedAt,
		&organizationName, &roleName,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("пользователь не найден")
	}
	if err != nil {
		log.Printf("❌ Ошибка при получении пользователя по ID: %v", err)
		return nil, err
	}

	// Конвертируем NULL значения
	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}
	if organizationID.Valid {
		user.OrganizationID = &organizationID.String
	}
	if organizationName.Valid {
		user.OrganizationName = organizationName.String
	}
	if roleName.Valid {
		user.RoleName = roleName.String
	}
	if createdBy.Valid { // Используем createdBy
		user.CreatedBy = &createdBy.String
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return user, nil
}

// GetAll - получает всех пользователей с фильтрацией
func (r *UserRepository) GetAll(roleID int, organizationID string) ([]*models.User, error) {
	var rows *sql.Rows
	var err error

	query := `
        SELECT 
            u.id, u.username, u.full_name, u.email, u.phone,
            u.organization_id, u.role_id, u.is_active, u.last_login,
            u.created_at,
            o.name as organization_name,
            r.name as role_name
        FROM users u
        LEFT JOIN organizations o ON u.organization_id = o.id
        LEFT JOIN roles r ON u.role_id = r.id
        WHERE 1=1
    `

	var args []interface{}
	argCount := 1

	if roleID > 0 {
		query += " AND u.role_id = $" + string(rune(argCount+'0'))
		args = append(args, roleID)
		argCount++
	}

	if organizationID != "" {
		query += " AND u.organization_id = $" + string(rune(argCount+'0'))
		args = append(args, organizationID)
		argCount++
	}

	query += " ORDER BY u.created_at DESC"

	rows, err = r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var email, phone, organizationName, roleName sql.NullString
		var orgID sql.NullString
		var lastLogin sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Username, &user.FullName, &email, &phone,
			&orgID, &user.RoleID, &user.IsActive, &lastLogin,
			&user.CreatedAt, &organizationName, &roleName,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			user.Email = &email.String
		}
		if phone.Valid {
			user.Phone = &phone.String
		}
		if orgID.Valid {
			user.OrganizationID = &orgID.String
		}
		if organizationName.Valid {
			user.OrganizationName = organizationName.String
		}
		if roleName.Valid {
			user.RoleName = roleName.String
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}
	return users, nil
}

// GetByOrganization - получает всех пользователей организации
func (r *UserRepository) GetByOrganization(orgID string) ([]*models.User, error) {
	rows, err := r.db.Query(`
        SELECT 
            u.id, u.username, u.full_name, u.email, u.phone,
            u.role_id, u.is_active, u.last_login, u.created_at,
            r.name as role_name
        FROM users u
        LEFT JOIN roles r ON u.role_id = r.id
        WHERE u.organization_id = $1
        ORDER BY u.created_at DESC
    `, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var email, phone, roleName sql.NullString
		var lastLogin sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Username, &user.FullName, &email, &phone,
			&user.RoleID, &user.IsActive, &lastLogin, &user.CreatedAt,
			&roleName,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			user.Email = &email.String
		}
		if phone.Valid {
			user.Phone = &phone.String
		}
		if roleName.Valid {
			user.RoleName = roleName.String
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}
	return users, nil
}

// Update - обновляет данные пользователя
func (r *UserRepository) Update(user *models.User) error {
	query := `
        UPDATE users SET
            full_name = $1,
            email = $2,
            phone = $3,
            organization_id = $4,
            role_id = $5,
            is_active = $6,
            updated_at = $7
        WHERE id = $8
    `
	result, err := r.db.Exec(query,
		user.FullName, user.Email, user.Phone,
		user.OrganizationID, user.RoleID, user.IsActive,
		time.Now(), user.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

// UpdatePassword - обновляет пароль пользователя
func (r *UserRepository) UpdatePassword(userID, hashedPassword string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.Exec(query, hashedPassword, time.Now(), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

// UpdateLastLogin - обновляет время последнего входа
func (r *UserRepository) UpdateLastLogin(userID string) error {
	query := `UPDATE users SET last_login = $1 WHERE id = $2`
	_, err := r.db.Exec(query, time.Now(), userID)
	return err
}

// Delete - мягкое удаление (деактивация) пользователя
func (r *UserRepository) Delete(id string) error {
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE id = $2`
	result, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

// HardDelete - полное удаление пользователя из БД (только для админов)
func (r *UserRepository) HardDelete(id string) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

// CheckListPermission - проверяет, имеет ли пользователь право подавать заявку в список
func (r *UserRepository) CheckListPermission(userID, listID string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM user_list_permissions 
            WHERE user_id = $1 AND list_id = $2
        )
    `
	err := r.db.QueryRow(query, userID, listID).Scan(&exists)
	return exists, err
}

// GetUserListPermissions - получает все списки, доступные пользователю
// GetUserListPermissions - получает все списки, доступные пользователю
func (r *UserRepository) GetUserListPermissions(userID string) ([]*models.AccessList, error) {
	// Сначала проверим, есть ли у пользователя какие-либо права
	var count int
	err := r.db.QueryRow(`
        SELECT COUNT(*) FROM user_list_permissions WHERE user_id = $1
    `, userID).Scan(&count)

	if err != nil {
		log.Printf("❌ Ошибка при проверке прав пользователя: %v", err)
		// Если ошибка, возможно таблица user_list_permissions не существует
		// Вернем пустой список, но не ошибку
		return []*models.AccessList{}, nil
	}

	log.Printf("📊 Найдено %d прав для пользователя %s", count, userID)

	// Если нет прав, возвращаем пустой список
	if count == 0 {
		return []*models.AccessList{}, nil
	}

	// Получаем списки, на которые у пользователя есть права
	rows, err := r.db.Query(`
        SELECT 
            al.id, al.name, al.description, al.color, al.priority, al.is_active
        FROM access_lists al
        INNER JOIN user_list_permissions ulp ON al.id = ulp.list_id
        WHERE ulp.user_id = $1 AND al.is_active = true
        ORDER BY al.priority, al.name
    `, userID)

	if err != nil {
		log.Printf("❌ Ошибка при выполнении запроса списков: %v", err)
		return nil, err
	}
	defer rows.Close()

	var lists []*models.AccessList
	for rows.Next() {
		list := &models.AccessList{}
		var description sql.NullString
		var color sql.NullString

		err := rows.Scan(
			&list.ID, &list.Name, &description, &color,
			&list.Priority, &list.IsActive,
		)
		if err != nil {
			log.Printf("❌ Ошибка при сканировании строки: %v", err)
			continue
		}

		if description.Valid {
			list.Description = description.String
		}
		if color.Valid {
			list.Color = color.String
		}

		lists = append(lists, list)
	}

	// Проверяем ошибки после завершения rows.Next()
	if err = rows.Err(); err != nil {
		log.Printf("❌ Ошибка при итерации по строкам: %v", err)
		return nil, err
	}

	log.Printf("✅ Найдено %d доступных списков для пользователя %s", len(lists), userID)
	return lists, nil
}

// AddListPermission - добавляет пользователю право на список
func (r *UserRepository) AddListPermission(userID, listID string) error {
	query := `
        INSERT INTO user_list_permissions (user_id, list_id, created_at)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, list_id) DO NOTHING
    `
	_, err := r.db.Exec(query, userID, listID, time.Now())
	return err
}

// RemoveListPermission - удаляет у пользователя право на список
func (r *UserRepository) RemoveListPermission(userID, listID string) error {
	query := `DELETE FROM user_list_permissions WHERE user_id = $1 AND list_id = $2`
	_, err := r.db.Exec(query, userID, listID)
	return err
}

// GetUsersByRole - получает всех пользователей с определенной ролью
func (r *UserRepository) GetUsersByRole(roleID int) ([]*models.User, error) {
	rows, err := r.db.Query(`
        SELECT 
            u.id, u.username, u.full_name, u.email, u.phone,
            u.organization_id, u.is_active, u.last_login, u.created_at,
            o.name as organization_name
        FROM users u
        LEFT JOIN organizations o ON u.organization_id = o.id
        WHERE u.role_id = $1 AND u.is_active = true
        ORDER BY u.full_name
    `, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var email, phone, organizationName sql.NullString
		var orgID sql.NullString
		var lastLogin sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Username, &user.FullName, &email, &phone,
			&orgID, &user.IsActive, &lastLogin, &user.CreatedAt,
			&organizationName,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			user.Email = &email.String
		}
		if phone.Valid {
			user.Phone = &phone.String
		}
		if orgID.Valid {
			user.OrganizationID = &orgID.String
		}
		if organizationName.Valid {
			user.OrganizationName = organizationName.String
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}
	return users, nil
}

// Search - поиск пользователей по имени, email или username
func (r *UserRepository) Search(query string) ([]*models.User, error) {
	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(`
        SELECT 
            u.id, u.username, u.full_name, u.email, u.phone,
            u.organization_id, u.role_id, u.is_active, u.last_login, u.created_at,
            o.name as organization_name,
            r.name as role_name
        FROM users u
        LEFT JOIN organizations o ON u.organization_id = o.id
        LEFT JOIN roles r ON u.role_id = r.id
        WHERE 
            u.full_name ILIKE $1 OR 
            u.username ILIKE $1 OR 
            u.email ILIKE $1 OR
            u.phone ILIKE $1
        ORDER BY u.full_name
        LIMIT 20
    `, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var email, phone, organizationName, roleName sql.NullString
		var orgID, createdBy sql.NullString // createdBy объявлена здесь
		var lastLogin sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Username, &user.FullName, &email, &phone,
			&orgID, &user.RoleID, &user.IsActive, &lastLogin, &user.CreatedAt,
			&organizationName, &roleName,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			user.Email = &email.String
		}
		if phone.Valid {
			user.Phone = &phone.String
		}
		if orgID.Valid {
			user.OrganizationID = &orgID.String
		}
		if organizationName.Valid {
			user.OrganizationName = organizationName.String
		}
		if roleName.Valid {
			user.RoleName = roleName.String
		}
		if createdBy.Valid { // Используем createdBy
			user.CreatedBy = &createdBy.String
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}
	return users, nil
}

// Count - возвращает количество пользователей
func (r *UserRepository) Count(roleID int, organizationID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE 1=1`
	var args []interface{}
	argCount := 1

	if roleID > 0 {
		query += " AND role_id = $" + string(rune(argCount+'0'))
		args = append(args, roleID)
		argCount++
	}

	if organizationID != "" {
		query += " AND organization_id = $" + string(rune(argCount+'0'))
		args = append(args, organizationID)
		argCount++
	}

	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// GetByEmail - получает пользователя по email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT 
            id, username, full_name, email, phone,
            organization_id, role_id, is_active
        FROM users 
        WHERE email = $1 AND is_active = true
    `

	var userEmail, phone sql.NullString
	var orgID sql.NullString

	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.FullName,
		&userEmail, &phone, &orgID,
		&user.RoleID, &user.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("пользователь не найден")
	}
	if err != nil {
		return nil, err
	}

	if userEmail.Valid {
		user.Email = &userEmail.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}
	if orgID.Valid {
		user.OrganizationID = &orgID.String
	}

	return user, err
}

// GetByPhone - получает пользователя по телефону
func (r *UserRepository) GetByPhone(phone string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT 
            id, username, full_name, email, phone,
            organization_id, role_id, is_active
        FROM users 
        WHERE phone = $1 AND is_active = true
    `

	var userEmail, userPhone sql.NullString
	var orgID sql.NullString

	err := r.db.QueryRow(query, phone).Scan(
		&user.ID, &user.Username, &user.FullName,
		&userEmail, &userPhone, &orgID,
		&user.RoleID, &user.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("пользователь не найден")
	}
	if err != nil {
		return nil, err
	}

	if userEmail.Valid {
		user.Email = &userEmail.String
	}
	if userPhone.Valid {
		user.Phone = &userPhone.String
	}
	if orgID.Valid {
		user.OrganizationID = &orgID.String
	}

	return user, err
}

// CheckUsernameExists - проверяет существует ли username
func (r *UserRepository) CheckUsernameExists(username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	err := r.db.QueryRow(query, username).Scan(&exists)
	return exists, err
}

// CheckEmailExists - проверяет существует ли email
func (r *UserRepository) CheckEmailExists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}

// GetUsersWithPagination - получает пользователей с пагинацией
func (r *UserRepository) GetUsersWithPagination(offset, limit int, roleID int, organizationID string) ([]*models.User, error) {
	var rows *sql.Rows
	var err error

	query := `
        SELECT 
            u.id, u.username, u.full_name, u.email, u.phone,
            u.organization_id, u.role_id, u.is_active, u.last_login,
            u.created_at,
            o.name as organization_name,
            r.name as role_name
        FROM users u
        LEFT JOIN organizations o ON u.organization_id = o.id
        LEFT JOIN roles r ON u.role_id = r.id
        WHERE 1=1
    `

	var args []interface{}
	argCount := 1

	if roleID > 0 {
		query += " AND u.role_id = $" + string(rune(argCount+'0'))
		args = append(args, roleID)
		argCount++
	}

	if organizationID != "" {
		query += " AND u.organization_id = $" + string(rune(argCount+'0'))
		args = append(args, organizationID)
		argCount++
	}

	query += " ORDER BY u.created_at DESC LIMIT $" + string(rune(argCount+'0'))
	args = append(args, limit)
	argCount++

	query += " OFFSET $" + string(rune(argCount+'0'))
	args = append(args, offset)

	rows, err = r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var email, phone, organizationName, roleName sql.NullString
		var orgID sql.NullString
		var lastLogin sql.NullTime
		var createdBy sql.NullString // Добавляем createdBy

		err := rows.Scan(
			&user.ID, &user.Username, &user.FullName, &email, &phone,
			&orgID, &user.RoleID, &user.IsActive, &lastLogin,
			&user.CreatedAt, &organizationName, &roleName,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			user.Email = &email.String
		}
		if phone.Valid {
			user.Phone = &phone.String
		}
		if orgID.Valid {
			user.OrganizationID = &orgID.String
		}
		if organizationName.Valid {
			user.OrganizationName = organizationName.String
		}
		if roleName.Valid {
			user.RoleName = roleName.String
		}
		if createdBy.Valid { // Используем createdBy
			user.CreatedBy = &createdBy.String
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}
	return users, nil
}
