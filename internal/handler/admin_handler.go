package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

type AdminHandler struct {
	userRepo         *repository.UserRepository
	organizationRepo *repository.OrganizationRepository
	contractRepo     *repository.ContractRepository
	accessListRepo   *repository.AccessListRepository
	approvedRepo     *repository.ApprovedPlateRepository
	applicationRepo  *repository.ApplicationRepository // Добавляем
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	organizationRepo *repository.OrganizationRepository,
	contractRepo *repository.ContractRepository,
	accessListRepo *repository.AccessListRepository,
	approvedRepo *repository.ApprovedPlateRepository,
	applicationRepo *repository.ApplicationRepository, // Добавляем
) *AdminHandler {
	return &AdminHandler{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
		contractRepo:     contractRepo,
		accessListRepo:   accessListRepo,
		approvedRepo:     approvedRepo,
		applicationRepo:  applicationRepo, // Добавляем
	}
}

// ============== Организации ==============

// CreateOrganization - создание организации
func (h *AdminHandler) CreateOrganization(c *gin.Context) {
	var req models.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	// Проверяем уникальность БИН
	exists, err := h.organizationRepo.CheckBINExists(req.BIN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке БИН"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Организация с таким БИН уже существует"})
		return
	}

	org := &models.Organization{
		ID:           uuid.New().String(),
		Name:         req.Name,
		BIN:          req.BIN,
		Address:      req.Address,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.organizationRepo.Create(org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании организации"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// GetAllOrganizations - получение всех организаций
func (h *AdminHandler) GetAllOrganizations(c *gin.Context) {
	organizations, err := h.organizationRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении организаций"})
		return
	}

	c.JSON(http.StatusOK, organizations)
}

// GetOrganization - получение организации по ID
func (h *AdminHandler) GetOrganization(c *gin.Context) {
	id := c.Param("id")

	org, err := h.organizationRepo.GetWithStats(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Организация не найдена"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// UpdateOrganization - обновление организации
func (h *AdminHandler) UpdateOrganization(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	org := &models.Organization{
		ID:           id,
		Name:         req.Name,
		BIN:          req.BIN,
		Address:      req.Address,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		UpdatedAt:    time.Now(),
	}

	if err := h.organizationRepo.Update(org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении организации"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// DeleteOrganization - удаление организации
func (h *AdminHandler) DeleteOrganization(c *gin.Context) {
	id := c.Param("id")

	if err := h.organizationRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Организация удалена"})
}

// ============== Пользователи ==============

// CreateUser - создание пользователя
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	// Проверяем уникальность username
	exists, err := h.userRepo.CheckUsernameExists(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке username"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователь с таким логином уже существует"})
		return
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при хешировании пароля"})
		return
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)

	user := &models.User{
		ID:             uuid.New().String(),
		Username:       req.Username,
		PasswordHash:   string(hashedPassword),
		FullName:       req.FullName,
		Email:          req.Email,
		Phone:          req.Phone,
		OrganizationID: req.OrganizationID,
		RoleID:         req.RoleID,
		IsActive:       true,
		CreatedBy:      &adminIDStr,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	// Убираем пароль из ответа
	user.PasswordHash = ""
	c.JSON(http.StatusCreated, user)
}

// GetAllUsers - получение всех пользователей
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	users, err := h.userRepo.GetAll(0, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении пользователей"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser - получение пользователя по ID
func (h *AdminHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

// UpdateUser - обновление пользователя
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	user := &models.User{
		ID:             id,
		FullName:       req.FullName,
		Email:          req.Email,
		Phone:          req.Phone,
		OrganizationID: req.OrganizationID,
		RoleID:         req.RoleID,
		UpdatedAt:      time.Now(),
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пользователя"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser - удаление пользователя
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if err := h.userRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь удален"})
}

// ============== Договоры ==============

// CreateContract - создание договора
func (h *AdminHandler) CreateContract(c *gin.Context) {
	var req models.CreateContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	// Парсим даты
	contractDate, err := time.Parse("2006-01-02", req.ContractDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты договора"})
		return
	}

	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты начала"})
		return
	}

	var validUntil *time.Time
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты окончания"})
			return
		}
		validUntil = &t
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)

	contract := &models.Contract{
		ID:             uuid.New().String(),
		ContractNumber: req.ContractNumber,
		OrganizationID: req.OrganizationID,
		ContractDate:   contractDate,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		ContractType:   req.ContractType,
		Status:         "active",
		Notes:          req.Notes,
		CreatedBy:      &adminIDStr,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.contractRepo.Create(contract); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании договора"})
		return
	}

	c.JSON(http.StatusCreated, contract)
}

// GetAllContracts - получение всех договоров
func (h *AdminHandler) GetAllContracts(c *gin.Context) {
	contracts, err := h.contractRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении договоров"})
		return
	}

	c.JSON(http.StatusOK, contracts)
}

// GetContractsByOrganization - получение договоров организации
func (h *AdminHandler) GetContractsByOrganization(c *gin.Context) {
	orgID := c.Param("id")

	contracts, err := h.contractRepo.GetByOrganization(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении договоров"})
		return
	}

	c.JSON(http.StatusOK, contracts)
}

// UpdateContract - обновление договора
func (h *AdminHandler) UpdateContract(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID договора не указан"})
		return
	}

	var req struct {
		ContractNumber string `json:"contractNumber"`
		OrganizationID string `json:"organizationId"`
		ContractDate   string `json:"contractDate"`
		ValidFrom      string `json:"validFrom"`
		ValidUntil     string `json:"validUntil"`
		ContractType   string `json:"contractType"`
		Status         string `json:"status"`
		Notes          string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	log.Printf("📝 Обновление договора %s: %+v", id, req)

	// Получаем существующий договор
	contract, err := h.contractRepo.GetByID(id)
	if err != nil {
		log.Printf("❌ Договор %s не найден: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Договор не найден"})
		return
	}

	// Обновляем поля
	if req.ContractNumber != "" {
		contract.ContractNumber = req.ContractNumber
	}
	if req.OrganizationID != "" {
		contract.OrganizationID = req.OrganizationID
	}
	if req.ContractDate != "" {
		t, err := time.Parse("2006-01-02", req.ContractDate)
		if err == nil {
			contract.ContractDate = t
		} else {
			log.Printf("⚠️ Ошибка парсинга contractDate: %v", err)
		}
	}
	if req.ValidFrom != "" {
		t, err := time.Parse("2006-01-02", req.ValidFrom)
		if err == nil {
			contract.ValidFrom = t
		} else {
			log.Printf("⚠️ Ошибка парсинга validFrom: %v", err)
		}
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err == nil {
			contract.ValidUntil = &t
		} else {
			log.Printf("⚠️ Ошибка парсинга validUntil: %v", err)
		}
	}
	if req.ContractType != "" {
		contract.ContractType = req.ContractType
	}
	if req.Status != "" {
		contract.Status = req.Status
	}
	if req.Notes != "" {
		contract.Notes = req.Notes
	}

	contract.UpdatedAt = time.Now()

	if err := h.contractRepo.Update(contract); err != nil {
		log.Printf("❌ Ошибка при обновлении договора %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении договора"})
		return
	}

	log.Printf("✅ Договор %s успешно обновлен", id)
	c.JSON(http.StatusOK, contract)
}

// ============== Списки доступа ==============

// CreateAccessList - создание списка доступа
// CreateAccessList - создание списка доступа
func (h *AdminHandler) CreateAccessList(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Priority    int    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат данных",
		})
		return
	}

	log.Printf("📝 Создание списка доступа: name=%s, description=%s, color=%s, priority=%d",
		req.Name, req.Description, req.Color, req.Priority)

	adminID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}
	adminIDStr := adminID.(string)

	list := &models.AccessList{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Priority:    req.Priority,
		IsActive:    true,
		CreatedBy:   &adminIDStr,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.accessListRepo.Create(list); err != nil {
		log.Printf("❌ Ошибка при создании списка: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при создании списка",
		})
		return
	}

	log.Printf("✅ Список доступа создан: ID=%s", list.ID)
	c.JSON(http.StatusCreated, list)
}

// GetAllAccessLists - получение всех списков
func (h *AdminHandler) GetAllAccessLists(c *gin.Context) {
	lists, err := h.accessListRepo.GetListsWithPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списков"})
		return
	}

	c.JSON(http.StatusOK, lists)
}

// GetAccessList - получение списка по ID
func (h *AdminHandler) GetAccessList(c *gin.Context) {
	id := c.Param("id")

	list, err := h.accessListRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Список не найден"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// UpdateAccessList - обновление списка
func (h *AdminHandler) UpdateAccessList(c *gin.Context) {
	id := c.Param("id")

	var req models.CreateAccessListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	list := &models.AccessList{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Priority:    req.Priority,
		UpdatedAt:   time.Now(),
	}

	if err := h.accessListRepo.Update(list); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении списка"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// DeleteAccessList - удаление списка
func (h *AdminHandler) DeleteAccessList(c *gin.Context) {
	id := c.Param("id")

	if err := h.accessListRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении списка"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Список удален"})
}

// ============== Права на списки ==============

// AddListPermission - добавление права на список
func (h *AdminHandler) AddListPermission(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		ListID string `json:"listId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	if err := h.userRepo.AddListPermission(userID, req.ListID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении права"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Право добавлено"})
}

// GetUserListPermissions - получение прав пользователя
func (h *AdminHandler) GetUserListPermissions(c *gin.Context) {
	userID := c.Param("id")

	lists, err := h.userRepo.GetUserListPermissions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении прав"})
		return
	}

	c.JSON(http.StatusOK, lists)
}

// RemoveListPermission - удаление права на список
func (h *AdminHandler) RemoveListPermission(c *gin.Context) {
	userID := c.Param("id")
	listID := c.Param("listId")

	if err := h.userRepo.RemoveListPermission(userID, listID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении права"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Право удалено"})
}

// ============== Прямое добавление номеров ==============

// AddDirectPlate - прямое добавление номера в список
// AddDirectPlate - прямое добавление номера в список
// AddDirectPlate - прямое добавление номера в список
func (h *AdminHandler) AddDirectPlate(c *gin.Context) {
	var req models.CreateApprovedPlateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	log.Printf("📝 Прямое добавление номера: %s в список %s", req.PlateNumber, req.ListID)

	// Сначала проверяем, существует ли уже такой номер в этом списке (включая неактивные)
	existingPlate, err := h.approvedRepo.GetByPlateNumberAndListIncludeInactive(req.PlateNumber, req.ListID)
	if err == nil && existingPlate != nil {
		// Номер уже существует, но возможно неактивен
		if !existingPlate.IsActive {
			// Реактивируем существующий номер
			log.Printf("🔄 Найден неактивный номер %s, выполняем реактивацию", req.PlateNumber)

			existingPlate.IsActive = true
			existingPlate.UpdatedAt = time.Now()

			// Обновляем остальные поля, если они изменились
			if req.ValidFrom != "" {
				t, _ := time.Parse("2006-01-02", req.ValidFrom)
				existingPlate.ValidFrom = &t
			}
			if req.ValidUntil != "" {
				t, _ := time.Parse("2006-01-02", req.ValidUntil)
				existingPlate.ValidUntil = &t
			}
			if req.Notes != "" {
				existingPlate.Notes = req.Notes
			}
			if req.VehicleBrand != "" {
				existingPlate.VehicleBrand = req.VehicleBrand
			}
			if req.VehicleModel != "" {
				existingPlate.VehicleModel = req.VehicleModel
			}
			if req.VehicleColor != "" {
				existingPlate.VehicleColor = req.VehicleColor
			}

			// Сохраняем изменения
			if err := h.approvedRepo.Update(existingPlate); err != nil {
				log.Printf("❌ Ошибка при реактивации номера: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при реактивации номера"})
				return
			}

			log.Printf("✅ Номер %s успешно реактивирован", req.PlateNumber)
			c.JSON(http.StatusOK, existingPlate)
			return
		} else {
			// Номер активен - возвращаем ошибку
			log.Printf("⚠️ Активный номер %s уже существует в списке", req.PlateNumber)
			c.JSON(http.StatusConflict, gin.H{"error": "Такой номер уже есть в этом списке"})
			return
		}
	}

	// Если номера нет, создаем новый
	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)

	// Парсим даты
	var validFrom, validUntil *time.Time
	if req.ValidFrom != "" {
		t, err := time.Parse("2006-01-02", req.ValidFrom)
		if err == nil {
			validFrom = &t
		}
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err == nil {
			validUntil = &t
		}
	}

	plate := &models.ApprovedPlate{
		ID:             uuid.New().String(),
		PlateNumber:    req.PlateNumber,
		VehicleBrand:   req.VehicleBrand,
		VehicleModel:   req.VehicleModel,
		VehicleColor:   req.VehicleColor,
		OrganizationID: &req.OrganizationID,
		ListID:         req.ListID,
		ApprovedBy:     &adminIDStr,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		IsActive:       true,
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.approvedRepo.Create(plate); err != nil {
		log.Printf("❌ Ошибка при добавлении номера в БД: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении номера"})
		return
	}

	log.Printf("✅ Номер %s успешно добавлен в список %s", plate.PlateNumber, plate.ListID)
	c.JSON(http.StatusCreated, plate)
}

// GetAllApprovedPlates - получение всех утвержденных номеров (включая неактивные)
func (h *AdminHandler) GetAllApprovedPlates(c *gin.Context) {
	// Получаем параметр фильтрации по активности из запроса
	onlyActive := c.Query("active") == "true"

	plates, err := h.approvedRepo.GetAll("", "", onlyActive)
	if err != nil {
		log.Printf("❌ Ошибка при получении номеров: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}

	c.JSON(http.StatusOK, plates)
}

// GetApprovedPlatesByList - получение номеров по списку
func (h *AdminHandler) GetApprovedPlatesByList(c *gin.Context) {
	listID := c.Param("listId")

	plates, err := h.approvedRepo.GetByList(listID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}

	c.JSON(http.StatusOK, plates)
}

// RemoveApprovedPlate - полное удаление номера из списка
func (h *AdminHandler) RemoveApprovedPlate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID номера не указан"})
		return
	}

	log.Printf("🗑️ Полное удаление номера %s", id)

	// Полное удаление из базы данных
	if err := h.approvedRepo.HardDelete(id); err != nil {
		log.Printf("❌ Ошибка при удалении номера %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении номера"})
		return
	}

	log.Printf("✅ Номер %s полностью удален из базы данных", id)
	c.JSON(http.StatusOK, gin.H{"message": "Номер полностью удален из базы данных"})
}

// DeactivateApprovedPlate - мягкое удаление (деактивация) номера
func (h *AdminHandler) DeactivateApprovedPlate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID номера не указан"})
		return
	}

	log.Printf("🔒 Деактивация номера %s", id)

	if err := h.approvedRepo.Delete(id); err != nil {
		log.Printf("❌ Ошибка при деактивации номера %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при деактивации номера"})
		return
	}

	log.Printf("✅ Номер %s деактивирован", id)
	c.JSON(http.StatusOK, gin.H{"message": "Номер деактивирован"})
}

// ============== Статистика ==============

// GetDashboardStats - получение статистики для дашборда админа
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	// Получаем количество организаций
	orgs, err := h.organizationRepo.GetAll()
	orgsCount := 0
	if err == nil {
		orgsCount = len(orgs)
	}

	// Получаем количество пользователей
	users, err := h.userRepo.GetAll(0, "")
	usersCount := 0
	if err == nil {
		usersCount = len(users)
	}

	// Получаем количество активных договоров
	contracts, err := h.contractRepo.GetAll()
	activeContracts := 0
	if err == nil {
		for _, c := range contracts {
			if c.Status == "active" {
				activeContracts++
			}
		}
	}

	// Получаем количество утвержденных номеров
	plates, err := h.approvedRepo.GetAll("", "", true)
	platesCount := 0
	if err == nil {
		platesCount = len(plates)
	}

	// Получаем количество заявок по статусам
	// Это можно реализовать через отдельный метод в applicationRepo

	c.JSON(http.StatusOK, gin.H{
		"organizations_count": orgsCount,
		"users_count":         usersCount,
		"active_contracts":    activeContracts,
		"approved_plates":     platesCount,
		"total_contracts":     len(contracts),
	})
}

// UpdateUserPassword - смена пароля пользователя (только для админа)
func (h *AdminHandler) UpdateUserPassword(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID пользователя не указан"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных. Пароль должен содержать минимум 6 символов"})
		return
	}

	log.Printf("🔐 Смена пароля для пользователя: %s", userID)

	// Хешируем новый пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Ошибка при хешировании пароля: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке пароля"})
		return
	}

	// Обновляем пароль в базе данных
	err = h.userRepo.UpdatePassword(userID, string(hashedPassword))
	if err != nil {
		log.Printf("❌ Ошибка при обновлении пароля: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	log.Printf("✅ Пароль успешно изменен для пользователя: %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменен"})
}

// DeleteContract - удаление договора
func (h *AdminHandler) DeleteContract(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID договора не указан"})
		return
	}

	log.Printf("📝 Удаление договора: %s", id)

	// Проверяем, есть ли связанные заявки
	applications, err := h.applicationRepo.GetByContract(id)
	if err != nil {
		log.Printf("❌ Ошибка при проверке связанных заявок: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке связанных заявок"})
		return
	}

	if len(applications) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить договор, к которому привязаны заявки"})
		return
	}

	// Проверяем, есть ли связанные утвержденные номера
	plates, err := h.approvedRepo.GetByContract(id)
	if err != nil {
		log.Printf("❌ Ошибка при проверке связанных номеров: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке связанных номеров"})
		return
	}

	if len(plates) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить договор, к которому привязаны номера"})
		return
	}

	// Удаляем договор
	err = h.contractRepo.Delete(id)
	if err != nil {
		log.Printf("❌ Ошибка при удалении договора: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении договора"})
		return
	}

	log.Printf("✅ Договор %s успешно удален", id)
	c.JSON(http.StatusOK, gin.H{"message": "Договор успешно удален"})
}

// GetContractByID - получение договора по ID
func (h *AdminHandler) GetContractByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID договора не указан"})
		return
	}

	contract, err := h.contractRepo.GetByID(id)
	if err != nil {
		if err.Error() == "договор не найден" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Договор не найден"})
		} else {
			log.Printf("❌ Ошибка при получении договора %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении договора"})
		}
		return
	}

	c.JSON(http.StatusOK, contract)
}

// UpdateApprovedPlate - обновление утвержденного номера
func (h *AdminHandler) UpdateApprovedPlate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID номера не указан"})
		return
	}

	var req struct {
		PlateNumber  string `json:"plateNumber"`
		VehicleBrand string `json:"vehicleBrand"`
		VehicleModel string `json:"vehicleModel"`
		VehicleColor string `json:"vehicleColor"`
		ListID       string `json:"listId"`
		ValidFrom    string `json:"validFrom"`
		ValidUntil   string `json:"validUntil"`
		Notes        string `json:"notes"`
		IsActive     *bool  `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	log.Printf("📝 Обновление номера %s", id)

	// Получаем существующий номер
	plate, err := h.approvedRepo.GetByID(id)
	if err != nil {
		log.Printf("❌ Номер %s не найден: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Номер не найден"})
		return
	}

	// Обновляем поля
	if req.PlateNumber != "" {
		// Проверяем уникальность номера в списке
		exists, err := h.approvedRepo.CheckIfExists(req.PlateNumber, plate.ListID)
		if err != nil {
			log.Printf("❌ Ошибка при проверке уникальности: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке номера"})
			return
		}
		if exists && req.PlateNumber != plate.PlateNumber {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Такой номер уже существует в этом списке"})
			return
		}
		plate.PlateNumber = req.PlateNumber
	}

	if req.VehicleBrand != "" {
		plate.VehicleBrand = req.VehicleBrand
	}
	if req.VehicleModel != "" {
		plate.VehicleModel = req.VehicleModel
	}
	if req.VehicleColor != "" {
		plate.VehicleColor = req.VehicleColor
	}
	if req.ListID != "" {
		plate.ListID = req.ListID
	}
	if req.ValidFrom != "" {
		t, err := time.Parse("2006-01-02", req.ValidFrom)
		if err == nil {
			plate.ValidFrom = &t
		}
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err == nil {
			plate.ValidUntil = &t
		}
	}
	if req.Notes != "" {
		plate.Notes = req.Notes
	}
	if req.IsActive != nil {
		plate.IsActive = *req.IsActive
	}

	plate.UpdatedAt = time.Now()

	if err := h.approvedRepo.Update(plate); err != nil {
		log.Printf("❌ Ошибка при обновлении номера %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении номера"})
		return
	}

	log.Printf("✅ Номер %s успешно обновлен", id)
	c.JSON(http.StatusOK, plate)
}
