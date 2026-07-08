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
	applicationRepo  *repository.ApplicationRepository
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	organizationRepo *repository.OrganizationRepository,
	contractRepo *repository.ContractRepository,
	accessListRepo *repository.AccessListRepository,
	approvedRepo *repository.ApprovedPlateRepository,
	applicationRepo *repository.ApplicationRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
		contractRepo:     contractRepo,
		accessListRepo:   accessListRepo,
		approvedRepo:     approvedRepo,
		applicationRepo:  applicationRepo,
	}
}

// ============== Организации ==============

func (h *AdminHandler) CreateOrganization(c *gin.Context) {
	var req models.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

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

func (h *AdminHandler) GetAllOrganizations(c *gin.Context) {
	organizations, err := h.organizationRepo.GetAll()
	if err != nil {
		log.Printf("❌ Ошибка при получении организаций: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении организаций"})
		return
	}
	c.JSON(http.StatusOK, organizations)
}

func (h *AdminHandler) GetOrganization(c *gin.Context) {
	id := c.Param("id")
	org, err := h.organizationRepo.GetWithStats(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Организация не найдена"})
		return
	}
	c.JSON(http.StatusOK, org)
}

func (h *AdminHandler) UpdateOrganization(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	org := &models.Organization{
		ID: id, Name: req.Name, BIN: req.BIN, Address: req.Address,
		ContactPhone: req.ContactPhone, ContactEmail: req.ContactEmail, UpdatedAt: time.Now(),
	}
	if err := h.organizationRepo.Update(org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении организации"})
		return
	}
	c.JSON(http.StatusOK, org)
}

func (h *AdminHandler) DeleteOrganization(c *gin.Context) {
	id := c.Param("id")
	if err := h.organizationRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Организация удалена"})
}

// ============== Пользователи ==============

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	exists, err := h.userRepo.CheckUsernameExists(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке username"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователь с таким логином уже существует"})
		return
	}

	// Проверка: для участника (roleId=4) — только один пользователь на организацию
	if req.RoleID == 4 && req.OrganizationID != nil {
		usersInOrg, err := h.userRepo.GetByOrganization(*req.OrganizationID)
		if err == nil {
			for _, u := range usersInOrg {
				if u.RoleID == 4 && u.IsActive {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "У данной организации уже есть пользователь с ролью Участник",
					})
					return
				}
			}
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при хешировании пароля"})
		return
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)

	user := &models.User{
		ID: uuid.New().String(), Username: req.Username, PasswordHash: string(hashedPassword),
		FullName: req.FullName, Email: req.Email, Phone: req.Phone,
		OrganizationID: req.OrganizationID, RoleID: req.RoleID, IsActive: true,
		CreatedBy: &adminIDStr, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusCreated, user)
}

func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	users, err := h.userRepo.GetAll(0, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении пользователей"})
		return
	}
	c.JSON(http.StatusOK, users)
}

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

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	user := &models.User{
		ID: id, FullName: req.FullName, Email: req.Email, Phone: req.Phone,
		OrganizationID: req.OrganizationID, RoleID: req.RoleID, UpdatedAt: time.Now(),
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

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.userRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении пользователя"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Пользователь удален"})
}

func (h *AdminHandler) HardDeleteUser(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)
	if id == adminIDStr {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить самого себя"})
		return
	}
	if err := h.userRepo.HardDelete(id); err != nil {
		log.Printf("❌ Ошибка при удалении пользователя: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении пользователя"})
		return
	}
	log.Printf("✅ Пользователь %s полностью удален", id)
	c.JSON(http.StatusOK, gin.H{"message": "Пользователь полностью удален"})
}

func (h *AdminHandler) UpdateUserPassword(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных. Пароль должен содержать минимум 6 символов"})
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке пароля"})
		return
	}
	if err := h.userRepo.UpdatePassword(userID, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменен"})
}

// ============== Договоры ==============

func (h *AdminHandler) CreateContract(c *gin.Context) {
	var req models.CreateContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	contractDate, _ := time.Parse("2006-01-02", req.ContractDate)
	validFrom, _ := time.Parse("2006-01-02", req.ValidFrom)
	var validUntil *time.Time
	if req.ValidUntil != "" {
		t, _ := time.Parse("2006-01-02", req.ValidUntil)
		validUntil = &t
	}
	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)

	contract := &models.Contract{
		ID: uuid.New().String(), ContractNumber: req.ContractNumber, OrganizationID: req.OrganizationID,
		ContractDate: contractDate, ValidFrom: validFrom, ValidUntil: validUntil,
		ContractType: req.ContractType, Status: "active", Notes: req.Notes,
		CreatedBy: &adminIDStr, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := h.contractRepo.Create(contract); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании договора"})
		return
	}
	c.JSON(http.StatusCreated, contract)
}

func (h *AdminHandler) GetAllContracts(c *gin.Context) {
	contracts, err := h.contractRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении договоров"})
		return
	}
	c.JSON(http.StatusOK, contracts)
}

func (h *AdminHandler) GetContractsByOrganization(c *gin.Context) {
	contracts, err := h.contractRepo.GetByOrganization(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении договоров"})
		return
	}
	c.JSON(http.StatusOK, contracts)
}

func (h *AdminHandler) GetContractByID(c *gin.Context) {
	id := c.Param("id")
	contract, err := h.contractRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Договор не найден"})
		return
	}
	c.JSON(http.StatusOK, contract)
}

func (h *AdminHandler) UpdateContract(c *gin.Context) {
	id := c.Param("id")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	contract, err := h.contractRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Договор не найден"})
		return
	}
	if req.ContractNumber != "" {
		contract.ContractNumber = req.ContractNumber
	}
	if req.OrganizationID != "" {
		contract.OrganizationID = req.OrganizationID
	}
	if req.ContractDate != "" {
		t, _ := time.Parse("2006-01-02", req.ContractDate)
		contract.ContractDate = t
	}
	if req.ValidFrom != "" {
		t, _ := time.Parse("2006-01-02", req.ValidFrom)
		contract.ValidFrom = t
	}
	if req.ValidUntil != "" {
		t, _ := time.Parse("2006-01-02", req.ValidUntil)
		contract.ValidUntil = &t
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении договора"})
		return
	}
	c.JSON(http.StatusOK, contract)
}

func (h *AdminHandler) DeleteContract(c *gin.Context) {
	id := c.Param("id")
	applications, _ := h.applicationRepo.GetByContract(id)
	if len(applications) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить договор, к которому привязаны заявки"})
		return
	}
	plates, _ := h.approvedRepo.GetByContract(id)
	if len(plates) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить договор, к которому привязаны номера"})
		return
	}
	if err := h.contractRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении договора"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Договор успешно удален"})
}

// ============== Списки доступа ==============

func (h *AdminHandler) CreateAccessList(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Priority    int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)
	list := &models.AccessList{
		ID: uuid.New().String(), Name: req.Name, Description: req.Description,
		Color: req.Color, Priority: req.Priority, IsActive: true,
		CreatedBy: &adminIDStr, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := h.accessListRepo.Create(list); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании списка"})
		return
	}
	c.JSON(http.StatusCreated, list)
}

func (h *AdminHandler) GetAllAccessLists(c *gin.Context) {
	lists, err := h.accessListRepo.GetListsWithPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списков"})
		return
	}
	c.JSON(http.StatusOK, lists)
}

func (h *AdminHandler) GetAccessList(c *gin.Context) {
	list, err := h.accessListRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Список не найден"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AdminHandler) UpdateAccessList(c *gin.Context) {
	id := c.Param("id")
	var req models.CreateAccessListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	list := &models.AccessList{
		ID: id, Name: req.Name, Description: req.Description,
		Color: req.Color, Priority: req.Priority, UpdatedAt: time.Now(),
	}
	if err := h.accessListRepo.Update(list); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении списка"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AdminHandler) DeleteAccessList(c *gin.Context) {
	if err := h.accessListRepo.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении списка"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Список удален"})
}

func (h *AdminHandler) HardDeleteAccessList(c *gin.Context) {
	id := c.Param("id")
	if err := h.accessListRepo.HardDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении списка"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Список полностью удален"})
}

// ============== Права на списки ==============

func (h *AdminHandler) AddListPermission(c *gin.Context) {
	var req struct {
		ListID string `json:"listId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	if err := h.userRepo.AddListPermission(c.Param("id"), req.ListID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении права"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Право добавлено"})
}

func (h *AdminHandler) GetUserListPermissions(c *gin.Context) {
	lists, err := h.userRepo.GetUserListPermissions(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении прав"})
		return
	}
	c.JSON(http.StatusOK, lists)
}

func (h *AdminHandler) RemoveListPermission(c *gin.Context) {
	if err := h.userRepo.RemoveListPermission(c.Param("id"), c.Param("listId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении права"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Право удалено"})
}

// ============== Утвержденные номера ==============

func (h *AdminHandler) AddDirectPlate(c *gin.Context) {
	var req models.CreateApprovedPlateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	existingPlate, err := h.approvedRepo.GetByPlateNumberAndListIncludeInactive(req.PlateNumber, req.ListID)
	if err == nil && existingPlate != nil {
		if !existingPlate.IsActive {
			existingPlate.IsActive = true
			existingPlate.UpdatedAt = time.Now()
			if req.VehicleBrand != "" {
				existingPlate.VehicleBrand = req.VehicleBrand
			}
			if req.VehicleModel != "" {
				existingPlate.VehicleModel = req.VehicleModel
			}
			if req.VehicleColor != "" {
				existingPlate.VehicleColor = req.VehicleColor
			}
			if req.Notes != "" {
				existingPlate.Notes = req.Notes
			}
			if req.ValidUntil != "" {
				t, _ := time.Parse("2006-01-02", req.ValidUntil)
				existingPlate.ValidUntil = &t
			}
			if err := h.approvedRepo.Update(existingPlate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при реактивации номера"})
				return
			}
			c.JSON(http.StatusOK, existingPlate)
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "Такой номер уже есть в этом списке"})
		return
	}

	// Парсим даты
	var validFrom, validUntil *time.Time
	if req.ValidFrom != "" {
		t, _ := time.Parse("2006-01-02", req.ValidFrom)
		validFrom = &t
	}
	if req.ValidUntil != "" {
		t, _ := time.Parse("2006-01-02", req.ValidUntil)
		validUntil = &t
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)
	plate := &models.ApprovedPlate{
		ID: uuid.New().String(), PlateNumber: req.PlateNumber, VehicleBrand: req.VehicleBrand,
		VehicleModel: req.VehicleModel, VehicleColor: req.VehicleColor,
		OrganizationID: &req.OrganizationID, ListID: req.ListID, ApprovedBy: &adminIDStr,
		ValidFrom: validFrom, ValidUntil: validUntil,
		IsActive: true, Notes: req.Notes, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := h.approvedRepo.Create(plate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении номера"})
		return
	}
	c.JSON(http.StatusCreated, plate)
}

func (h *AdminHandler) GetAllApprovedPlates(c *gin.Context) {
	onlyActive := c.Query("active") == "true"
	plates, err := h.approvedRepo.GetAll("", "", onlyActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}
	c.JSON(http.StatusOK, plates)
}

func (h *AdminHandler) GetApprovedPlatesByList(c *gin.Context) {
	plates, err := h.approvedRepo.GetByList(c.Param("listId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}
	c.JSON(http.StatusOK, plates)
}

func (h *AdminHandler) RemoveApprovedPlate(c *gin.Context) {
	if err := h.approvedRepo.HardDelete(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении номера"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Номер полностью удален из базы данных"})
}

func (h *AdminHandler) UpdateApprovedPlate(c *gin.Context) {
	id := c.Param("id")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}
	plate, err := h.approvedRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Номер не найден"})
		return
	}
	if req.PlateNumber != "" {
		exists, _ := h.approvedRepo.CheckIfExists(req.PlateNumber, plate.ListID)
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
		t, _ := time.Parse("2006-01-02", req.ValidFrom)
		plate.ValidFrom = &t
	}
	if req.ValidUntil != "" {
		t, _ := time.Parse("2006-01-02", req.ValidUntil)
		plate.ValidUntil = &t
	}
	if req.Notes != "" {
		plate.Notes = req.Notes
	}
	if req.IsActive != nil {
		plate.IsActive = *req.IsActive
	}
	plate.UpdatedAt = time.Now()
	if err := h.approvedRepo.Update(plate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении номера"})
		return
	}
	c.JSON(http.StatusOK, plate)
}

// ============== Статистика ==============

func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	orgs, _ := h.organizationRepo.GetAll()
	users, _ := h.userRepo.GetAll(0, "")
	contracts, _ := h.contractRepo.GetAll()
	plates, _ := h.approvedRepo.GetAll("", "", true)

	activeContracts := 0
	for _, c := range contracts {
		if c.Status == "active" {
			activeContracts++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"organizations_count": len(orgs),
		"users_count":         len(users),
		"active_contracts":    activeContracts,
		"approved_plates":     len(plates),
		"total_contracts":     len(contracts),
	})
}
func (h *AdminHandler) GetUsersByOrganization(c *gin.Context) {
	orgID := c.Param("id")
	users, err := h.userRepo.GetByOrganization(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении пользователей"})
		return
	}
	c.JSON(http.StatusOK, users)
}
