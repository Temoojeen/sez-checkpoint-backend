package repository

import (
	"database/sql"
	"log"
	"time"

	"sez-checkpoint-backend/internal/models"
)

type DeletedPlateRepository struct {
	db *sql.DB
}

func NewDeletedPlateRepository(db *sql.DB) *DeletedPlateRepository {
	return &DeletedPlateRepository{db: db}
}

// Create - создает запись об удаленном номере
func (r *DeletedPlateRepository) Create(plate *models.DeletedPlate) error {
	query := `
        INSERT INTO deleted_plates (
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            organization_id, organization_name, list_id, list_name,
            deleted_by, deleted_by_name, delete_reason, original_plate_id,
            created_at, deleted_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
    `
	_, err := r.db.Exec(query,
		plate.ID, plate.PlateNumber, plate.VehicleBrand, plate.VehicleModel, plate.VehicleColor,
		plate.OrganizationID, plate.OrganizationName, plate.ListID, plate.ListName,
		plate.DeletedBy, plate.DeletedByName, plate.DeleteReason, plate.OriginalPlateID,
		time.Now(), time.Now(),
	)
	if err != nil {
		log.Printf("❌ Ошибка при создании записи удаленного номера: %v", err)
	}
	return err
}

// GetAll - получает все удаленные номера
func (r *DeletedPlateRepository) GetAll() ([]*models.DeletedPlate, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            organization_id, organization_name, list_id, list_name,
            deleted_by, deleted_by_name, delete_reason, original_plate_id,
            created_at, deleted_at
        FROM deleted_plates
        ORDER BY deleted_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plates []*models.DeletedPlate
	for rows.Next() {
		plate := &models.DeletedPlate{}
		var vehicleBrand, vehicleModel, vehicleColor sql.NullString
		var organizationID, organizationName sql.NullString
		var listName sql.NullString
		var deletedBy, deletedByName, deleteReason sql.NullString
		var originalPlateID sql.NullString

		err := rows.Scan(
			&plate.ID, &plate.PlateNumber, &vehicleBrand, &vehicleModel, &vehicleColor,
			&organizationID, &organizationName, &plate.ListID, &listName,
			&deletedBy, &deletedByName, &deleteReason, &originalPlateID,
			&plate.CreatedAt, &plate.DeletedAt,
		)
		if err != nil {
			log.Printf("❌ Ошибка сканирования удаленного номера: %v", err)
			continue
		}

		if vehicleBrand.Valid {
			plate.VehicleBrand = vehicleBrand.String
		}
		if vehicleModel.Valid {
			plate.VehicleModel = vehicleModel.String
		}
		if vehicleColor.Valid {
			plate.VehicleColor = vehicleColor.String
		}
		if organizationID.Valid {
			plate.OrganizationID = &organizationID.String
		}
		if organizationName.Valid {
			plate.OrganizationName = organizationName.String
		}
		if listName.Valid {
			plate.ListName = listName.String
		}
		if deletedBy.Valid {
			plate.DeletedBy = &deletedBy.String
		}
		if deletedByName.Valid {
			plate.DeletedByName = deletedByName.String
		}
		if deleteReason.Valid {
			plate.DeleteReason = deleteReason.String
		}
		if originalPlateID.Valid {
			plate.OriginalPlateID = &originalPlateID.String
		}

		plates = append(plates, plate)
	}
	return plates, nil
}

// GetByOrganization - получает удаленные номера по организации
func (r *DeletedPlateRepository) GetByOrganization(organizationID string) ([]*models.DeletedPlate, error) {
	rows, err := r.db.Query(`
        SELECT 
            id, plate_number, vehicle_brand, vehicle_model, vehicle_color,
            organization_id, organization_name, list_id, list_name,
            deleted_by, deleted_by_name, delete_reason, original_plate_id,
            created_at, deleted_at
        FROM deleted_plates
        WHERE organization_id = $1
        ORDER BY deleted_at DESC
    `, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plates []*models.DeletedPlate
	for rows.Next() {
		plate := &models.DeletedPlate{}
		var vehicleBrand, vehicleModel, vehicleColor sql.NullString
		var orgID, organizationName sql.NullString
		var listName sql.NullString
		var deletedBy, deletedByName, deleteReason sql.NullString
		var originalPlateID sql.NullString

		err := rows.Scan(
			&plate.ID, &plate.PlateNumber, &vehicleBrand, &vehicleModel, &vehicleColor,
			&orgID, &organizationName, &plate.ListID, &listName,
			&deletedBy, &deletedByName, &deleteReason, &originalPlateID,
			&plate.CreatedAt, &plate.DeletedAt,
		)
		if err != nil {
			log.Printf("❌ Ошибка сканирования удаленного номера: %v", err)
			continue
		}

		if vehicleBrand.Valid {
			plate.VehicleBrand = vehicleBrand.String
		}
		if vehicleModel.Valid {
			plate.VehicleModel = vehicleModel.String
		}
		if vehicleColor.Valid {
			plate.VehicleColor = vehicleColor.String
		}
		if orgID.Valid {
			plate.OrganizationID = &orgID.String
		}
		if organizationName.Valid {
			plate.OrganizationName = organizationName.String
		}
		if listName.Valid {
			plate.ListName = listName.String
		}
		if deletedBy.Valid {
			plate.DeletedBy = &deletedBy.String
		}
		if deletedByName.Valid {
			plate.DeletedByName = deletedByName.String
		}
		if deleteReason.Valid {
			plate.DeleteReason = deleteReason.String
		}
		if originalPlateID.Valid {
			plate.OriginalPlateID = &originalPlateID.String
		}

		plates = append(plates, plate)
	}
	return plates, nil
}
