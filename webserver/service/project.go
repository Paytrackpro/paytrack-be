package service

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) CreateNewProject(userId uint64, projectRequest portal.ProjectRequest) (*storage.Project, error) {
	newProject := storage.Project{
		ProjectName: projectRequest.ProjectName,
		Members:     projectRequest.Members,
		CreatorId:   userId,
		Status:      storage.ProjectConfirmed,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to DB
	tx := s.db.Begin()
	// save new data
	if err := tx.Create(&newProject).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	return &newProject, nil
}

func (s *Service) GetMyProjects(userId uint64) ([]storage.Project, error) {
	projects := make([]storage.Project, 0)
	query := fmt.Sprintf(`SELECT * FROM projects WHERE status = %d AND members @> '[{"memberId": %d}]'`, storage.ProjectConfirmed, userId)
	if err := s.db.Raw(query).Scan(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *Service) UpdateProject(userId uint64, projectRequest portal.ProjectRequest) (storage.Project, error) {
	var project storage.Project
	if err := s.db.Where("project_id = ?", projectRequest.ProjectId).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return project, utils.NewError(fmt.Errorf("project not found"), utils.ErrorNotFound)
		}
		log.Error("UpdateProject:get project fail with error: ", err)
		return project, err
	}

	project.ProjectName = projectRequest.ProjectName
	project.Members = projectRequest.Members
	project.UpdatedAt = time.Now()

	tx := s.db.Begin()

	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProject:save project fail with error: ", err)
		return project, err
	}

	// update all related data
	payments := make([]*storage.Payment, 0)
	if err := s.db.Where("project_id = ?", projectRequest.ProjectId).Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return project, nil
		}
		return project, err
	}

	// validate payment
	for _, paym := range payments {
		paym.ProjectName = projectRequest.ProjectName
		if paym.Details != nil {
			details := make([]storage.PaymentDetail, 0)
			for _, detail := range paym.Details {
				if detail.ProjectId == projectRequest.ProjectId {
					detail.ProjectName = projectRequest.ProjectName
				}
				details = append(details, detail)
			}
			paym.Details = details
		}
	}

	if err := s.db.Save(&payments).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProject:update payment info fail with error: ", err)
		return project, err
	}

	tx.Commit()

	return project, nil
}
