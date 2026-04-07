package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"sez-checkpoint-backend/internal/handler"
	"sez-checkpoint-backend/internal/middleware"
	"sez-checkpoint-backend/internal/repository"
	"sez-checkpoint-backend/internal/websocket"
)

func main() {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Файл .env не найден, используем переменные окружения")
	}

	// Подключаемся к базе данных
	db, err := setupDatabase()
	if err != nil {
		log.Fatal("❌ Ошибка подключения к БД:", err)
	}
	defer db.Close()

	log.Println("✅ Подключение к БД успешно")

	// Создаем все таблицы
	if err := createAllTables(db); err != nil {
		log.Fatal("❌ Ошибка создания таблиц:", err)
	}

	// Создаем тестового админа с паролем 12346
	if err := createAdminUser(db); err != nil {
		log.Printf("⚠️  Ошибка при создании админа: %v", err)
	}

	// Создаем репозитории
	userRepo := repository.NewUserRepository(db)
	contractRepo := repository.NewContractRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	accessListRepo := repository.NewAccessListRepository(db)
	approvedPlateRepo := repository.NewApprovedPlateRepository(db)
	accessLogRepo := repository.NewAccessLogRepository(db)
	organizationRepo := repository.NewOrganizationRepository(db)

	// Создаем WebSocket хаб и запускаем его
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Println("✅ WebSocket хаб запущен")

	// Создаем хендлеры
	authHandler := handler.NewAuthHandler(userRepo, getJWTKey())
	userHandler := handler.NewUserHandler(userRepo, organizationRepo)
	applicationHandler := handler.NewApplicationHandler(
		applicationRepo,
		contractRepo,
		userRepo,
		approvedPlateRepo,
	)
	approvedPlateHandler := handler.NewApprovedPlateHandler(
		approvedPlateRepo,
		userRepo,
	)
	adminHandler := handler.NewAdminHandler(
		userRepo,
		organizationRepo,
		contractRepo,
		accessListRepo,
		approvedPlateRepo,
		applicationRepo,
	)
	securityHandler := handler.NewSecurityHandler(
		accessLogRepo,
		approvedPlateRepo,
	)

	// Создаем ANPR хендлер с WebSocket хабом
	anprHandler := handler.NewANPRHandler(
		accessLogRepo,
		approvedPlateRepo,
		wsHub, // Передаем WebSocket хаб
	)

	// Настраиваем роутер
	router := setupRouter(
		authHandler,
		userHandler,
		applicationHandler,
		approvedPlateHandler,
		adminHandler,
		securityHandler,
		anprHandler,
		wsHub, // Передаем WebSocket хаб для WebSocket endpoint
	)

	// Запускаем сервер
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Сервер sez-checkpoint-backend запущен на порту %s", port)
	log.Printf("🔐 Тестовый админ: логин=admin, пароль=12346")
	log.Printf("📸 ANPR endpoint: http://0.0.0.0:%s/api/camera-events", port)
	log.Printf("🔌 WebSocket endpoint: ws://0.0.0.0:%s/ws", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("❌ Ошибка запуска сервера:", err)
	}
}

// setupDatabase подключается к PostgreSQL
func setupDatabase() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "sez_checkpoint"
	}

	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	log.Printf("Подключение к БД: host=%s port=%s user=%s dbname=%s",
		host, port, user, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверяем подключение
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// createAdminUser создает пользователя admin с паролем 12346
func createAdminUser(db *sql.DB) error {
	// Проверяем, существует ли уже админ
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username = 'admin')`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("ошибка при проверке существования админа: %v", err)
	}

	if exists {
		log.Println("👤 Пользователь admin уже существует")
		return nil
	}

	// Хешируем пароль 12346
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("12346"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка при хешировании пароля: %v", err)
	}

	// Вставляем админа
	_, err = db.Exec(`
        INSERT INTO users (
            id, username, password_hash, full_name, role_id, is_active, created_at, updated_at
        ) VALUES (
            'ffffffff-ffff-ffff-ffff-ffffffffffff', 
            'admin', 
            $1, 
            'Главный администратор', 
            1, 
            true, 
            NOW(), 
            NOW()
        )
    `, string(hashedPassword))

	if err != nil {
		return fmt.Errorf("ошибка при создании админа: %v", err)
	}

	log.Println("✅ Пользователь admin успешно создан")
	return nil
}

// createAllTables создает все таблицы в базе данных
func createAllTables(db *sql.DB) error {
	// Включаем расширение для генерации UUID
	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		return fmt.Errorf("ошибка включения uuid-ossp: %v", err)
	}

	// Создаем таблицу организаций
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS organizations (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            name VARCHAR(255) NOT NULL,
            bin VARCHAR(12) UNIQUE,
            address TEXT,
            contact_phone VARCHAR(20),
            contact_email VARCHAR(100),
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы organizations: %v", err)
	}

	// Создаем таблицу ролей
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS roles (
            id SERIAL PRIMARY KEY,
            name VARCHAR(50) UNIQUE NOT NULL,
            description TEXT
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы roles: %v", err)
	}

	// Создаем таблицу договоров
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS contracts (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            contract_number VARCHAR(50) UNIQUE NOT NULL,
            organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
            contract_date DATE NOT NULL,
            valid_from DATE NOT NULL,
            valid_until DATE,
            contract_type VARCHAR(50) DEFAULT 'standard',
            status VARCHAR(20) DEFAULT 'active',
            file_path VARCHAR(500),
            notes TEXT,
            created_by UUID,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы contracts: %v", err)
	}

	// Создаем таблицу пользователей
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            username VARCHAR(100) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            full_name VARCHAR(255) NOT NULL,
            email VARCHAR(100),
            phone VARCHAR(20),
            organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
            role_id INTEGER REFERENCES roles(id),
            is_active BOOLEAN DEFAULT true,
            created_by UUID,
            last_login TIMESTAMP,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы users: %v", err)
	}

	// Создаем таблицу списков доступа
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS access_lists (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            name VARCHAR(100) NOT NULL,
            description TEXT,
            list_type VARCHAR(50) DEFAULT 'white',
            color VARCHAR(20),
            priority INTEGER DEFAULT 0,
            is_active BOOLEAN DEFAULT true,
            created_by UUID REFERENCES users(id),
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы access_lists: %v", err)
	}

	// Создаем таблицу разрешений пользователей на списки
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS user_list_permissions (
            user_id UUID REFERENCES users(id) ON DELETE CASCADE,
            list_id UUID REFERENCES access_lists(id) ON DELETE CASCADE,
            created_at TIMESTAMP DEFAULT NOW(),
            PRIMARY KEY (user_id, list_id)
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы user_list_permissions: %v", err)
	}

	// Создаем таблицу заявок
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS applications (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            plate_number VARCHAR(20) NOT NULL,
            vehicle_brand VARCHAR(50),
            vehicle_model VARCHAR(50),
            vehicle_color VARCHAR(30),
            contract_id UUID REFERENCES contracts(id),
            organization_id UUID REFERENCES organizations(id),
            list_id UUID REFERENCES access_lists(id),
            applicant_id UUID REFERENCES users(id),
            status VARCHAR(20) DEFAULT 'pending',
            operator_id UUID REFERENCES users(id),
            supervisor_id UUID REFERENCES users(id),
            operator_approved_at TIMESTAMP,
            supervisor_approved_at TIMESTAMP,
            rejected_at TIMESTAMP,
            reject_reason TEXT,
            valid_from DATE,
            valid_until DATE,
            notes TEXT,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы applications: %v", err)
	}

	// Создаем таблицу утвержденных номеров
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS approved_plates (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            plate_number VARCHAR(20) NOT NULL,
            vehicle_brand VARCHAR(50),
            vehicle_model VARCHAR(50),
            vehicle_color VARCHAR(30),
            contract_id UUID REFERENCES contracts(id),
            organization_id UUID REFERENCES organizations(id),
            list_id UUID REFERENCES access_lists(id),
            application_id UUID,
            approved_by UUID REFERENCES users(id),
            valid_from DATE DEFAULT CURRENT_DATE,
            valid_until DATE,
            is_active BOOLEAN DEFAULT true,
            notes TEXT,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW(),
            UNIQUE(plate_number, list_id)
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы approved_plates: %v", err)
	}

	// Создаем таблицу для прямого добавления номеров админом
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS direct_plate_additions (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            plate_number VARCHAR(20) NOT NULL,
            vehicle_brand VARCHAR(50),
            vehicle_model VARCHAR(50),
            vehicle_color VARCHAR(30),
            organization_id UUID REFERENCES organizations(id),
            list_id UUID REFERENCES access_lists(id),
            added_by UUID REFERENCES users(id),
            valid_from DATE DEFAULT CURRENT_DATE,
            valid_until DATE,
            notes TEXT,
            created_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы direct_plate_additions: %v", err)
	}

	// Создаем таблицу истории проездов
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS access_logs (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            plate_number VARCHAR(20) NOT NULL,
            organization_name VARCHAR(255),
            list_name VARCHAR(100),
            image_path VARCHAR(500),
            access_granted BOOLEAN DEFAULT true,
            camera_id VARCHAR(100),
            camera_location VARCHAR(255),
            created_at TIMESTAMP DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы access_logs: %v", err)
	}

	// Создаем индексы для быстрого поиска
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_organization ON users(organization_id)",
		"CREATE INDEX IF NOT EXISTS idx_contracts_number ON contracts(contract_number)",
		"CREATE INDEX IF NOT EXISTS idx_contracts_organization ON contracts(organization_id)",
		"CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(status)",
		"CREATE INDEX IF NOT EXISTS idx_applications_applicant ON applications(applicant_id)",
		"CREATE INDEX IF NOT EXISTS idx_applications_contract ON applications(contract_id)",
		"CREATE INDEX IF NOT EXISTS idx_approved_plates_number ON approved_plates(plate_number)",
		"CREATE INDEX IF NOT EXISTS idx_approved_plates_valid ON approved_plates(valid_from, valid_until)",
		"CREATE INDEX IF NOT EXISTS idx_access_logs_recent ON access_logs(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_access_logs_plate ON access_logs(plate_number)",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("ошибка создания индекса: %v", err)
		}
	}

	// Заполняем таблицу ролей начальными данными
	_, err = db.Exec(`
        INSERT INTO roles (id, name, description) VALUES 
            (1, 'admin', 'Полный доступ к системе'),
            (2, 'operator', 'Обработка заявок'),
            (3, 'supervisor', 'Финальное утверждение'),
            (4, 'participant', 'Подача заявок'),
            (5, 'security', 'Просмотр списков и истории')
        ON CONFLICT (id) DO NOTHING
    `)
	if err != nil {
		return fmt.Errorf("ошибка вставки ролей: %v", err)
	}

	log.Println("✅ Все таблицы успешно созданы")
	return nil
}

// getJWTKey возвращает секретный ключ для JWT
func getJWTKey() string {
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		key = "sez-checkpoint-super-secret-key-2024"
		log.Println("⚠️  JWT_SECRET не задан, используется ключ по умолчанию")
	}
	return key
}

// setupRouter настраивает все маршруты
func setupRouter(
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	applicationHandler *handler.ApplicationHandler,
	approvedPlateHandler *handler.ApprovedPlateHandler,
	adminHandler *handler.AdminHandler,
	securityHandler *handler.SecurityHandler,
	anprHandler *handler.ANPRHandler,
	wsHub *websocket.Hub,
) *gin.Engine {
	router := gin.Default()

	// Настройка CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://10.24.32.31:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// WebSocket endpoint (публичный)
	router.GET("/ws", wsHub.ServeWebSocket)

	// Публичный маршрут для камеры Hikvision (без JWT аутентификации)
	router.POST("/api/camera-events", anprHandler.HandleCameraEvent)

	// Публичные маршруты
	auth := router.Group("/api/auth")
	{
		auth.POST("/login", authHandler.Login)
	}

	// Защищенные маршруты (требуют JWT токен)
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware(getJWTKey()))
	{
		// Общие маршруты для всех авторизованных пользователей
		api.GET("/user/profile", userHandler.GetProfile)
		api.GET("/access-lists", userHandler.GetAvailableLists)
		api.GET("/user/list-permissions", userHandler.GetMyListPermissions)

		// Маршрут для получения номеров по списку (с проверкой прав)
		api.GET("/approved-plates/list/:listId", approvedPlateHandler.GetPlatesByList)

		// Маршруты для участника (roleId = 4)
		api.POST("/applications", middleware.RoleMiddleware(4), applicationHandler.Create)
		api.GET("/applications/my", middleware.RoleMiddleware(4), applicationHandler.GetMyApplications)

		// Маршруты для оператора (roleId = 2)
		api.GET("/applications/pending-operator", middleware.RoleMiddleware(2), applicationHandler.GetPendingForOperator)
		api.PUT("/applications/:id/operator-approve", middleware.RoleMiddleware(2), applicationHandler.OperatorApprove)
		api.PUT("/applications/:id/reject", middleware.RoleMiddleware(2), applicationHandler.Reject)

		// Маршруты для руководителя (roleId = 3)
		api.GET("/applications/pending-supervisor", middleware.RoleMiddleware(3), applicationHandler.GetPendingForSupervisor)
		api.PUT("/applications/:id/supervisor-approve", middleware.RoleMiddleware(3), applicationHandler.SupervisorApprove)

		// Общий маршрут для получения заявки по ID (доступен всем)
		api.GET("/applications/:id", applicationHandler.GetByID)

		// Маршруты для админа (roleId = 1)
		admin := api.Group("/admin")
		admin.Use(middleware.RoleMiddleware(1))
		{
			// Управление организациями
			admin.POST("/organizations", adminHandler.CreateOrganization)
			admin.GET("/organizations", adminHandler.GetAllOrganizations)
			admin.GET("/organizations/:id", adminHandler.GetOrganization)
			admin.PUT("/organizations/:id", adminHandler.UpdateOrganization)
			admin.DELETE("/organizations/:id", adminHandler.DeleteOrganization)

			// Управление договорами
			admin.POST("/contracts", adminHandler.CreateContract)
			admin.GET("/contracts", adminHandler.GetAllContracts)
			admin.GET("/organizations/:id/contracts", adminHandler.GetContractsByOrganization)
			admin.GET("/contracts/:id", adminHandler.GetContractByID)
			admin.PUT("/contracts/:id", adminHandler.UpdateContract)
			admin.DELETE("/contracts/:id", adminHandler.DeleteContract)

			// Управление пользователями
			admin.POST("/users", adminHandler.CreateUser)
			admin.GET("/users", adminHandler.GetAllUsers)
			admin.GET("/users/:id", adminHandler.GetUser)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)
			admin.PUT("/users/:id/password", adminHandler.UpdateUserPassword)

			// Управление списками доступа
			admin.POST("/access-lists", adminHandler.CreateAccessList)
			admin.GET("/access-lists", adminHandler.GetAllAccessLists)
			admin.GET("/access-lists/:id", adminHandler.GetAccessList)
			admin.PUT("/access-lists/:id", adminHandler.UpdateAccessList)
			admin.DELETE("/access-lists/:id", adminHandler.DeleteAccessList)
			admin.GET("/access-logs", securityHandler.GetAllLogs)

			// Права пользователей на списки
			admin.POST("/users/:id/list-permissions", adminHandler.AddListPermission)
			admin.GET("/users/:id/list-permissions", adminHandler.GetUserListPermissions)
			admin.DELETE("/users/:id/list-permissions/:listId", adminHandler.RemoveListPermission)

			// Управление утвержденными номерами
			admin.POST("/approved-plates/direct", adminHandler.AddDirectPlate)
			admin.GET("/approved-plates", adminHandler.GetAllApprovedPlates)
			admin.GET("/approved-plates/list/:listId", adminHandler.GetApprovedPlatesByList)
			admin.DELETE("/approved-plates/:id", adminHandler.RemoveApprovedPlate)
			admin.PUT("/approved-plates/:id", adminHandler.UpdateApprovedPlate)

			// Управление заявками (админские)
			admin.GET("/applications", applicationHandler.GetAllApplications)
			admin.PUT("/applications/:id/approve-as-operator", applicationHandler.AdminApproveAsOperator)
			admin.PUT("/applications/:id/approve-as-supervisor", applicationHandler.AdminApproveAsSupervisor)
			admin.PUT("/applications/:id/reject", applicationHandler.AdminReject)

			// Статистика
			admin.GET("/dashboard/stats", adminHandler.GetDashboardStats)
		}

		// Маршруты для охраны (roleId = 5)
		security := api.Group("/security")
		security.Use(middleware.RoleMiddleware(5))
		{
			security.GET("/recent-logs", securityHandler.GetRecentLogs)
			security.GET("/check-plate/:number", securityHandler.CheckPlate)
			security.POST("/log-access", securityHandler.LogAccess)
			security.GET("/statistics", securityHandler.GetStatistics)
			security.GET("/logs/plate/:number", securityHandler.GetLogsByPlate)
		}
	}

	return router
}
