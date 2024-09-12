package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) CreateNewProject(userId uint64, creatorName string, projectRequest portal.ProjectRequest) (*storage.Project, error) {
	newProject := storage.Project{
		ProjectName: projectRequest.ProjectName,
		Members:     projectRequest.Members,
		Approvers:   projectRequest.Approvers,
		Description: projectRequest.Description,
		CreatorId:   userId,
		CreatorName: creatorName,
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
	query := fmt.Sprintf(`SELECT * FROM projects WHERE status = %d AND (members @> '[{"memberId": %d}]' OR creator_id = %d)`, storage.ProjectConfirmed, userId, userId)
	if err := s.db.Raw(query).Scan(&projects).Error; err != nil {
		return nil, err
	}
	tx := s.db.Begin()
	result := make([]storage.Project, 0)
	for _, project := range projects {
		if project.CreatorId > 0 && utils.IsEmpty(project.CreatorName) {
			creator, err := s.GetUserInfo(project.CreatorId)
			if err == nil {
				project.CreatorName = creator.UserName
				if !utils.IsEmpty(creator.DisplayName) {
					project.CreatorName = creator.DisplayName
				}
				if err := tx.Save(&project).Error; err != nil {
					tx.Rollback()
					return projects, err
				}
			}
		}
		result = append(result, project)
	}
	tx.Commit()
	return result, nil
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
	project.Approvers = projectRequest.Approvers
	project.Description = projectRequest.Description
	if utils.IsEmpty(project.CreatorName) {
		userInfo, err := s.GetUserInfo(project.CreatorId)
		if err == nil {
			project.CreatorName = userInfo.UserName
			if !utils.IsEmpty(userInfo.DisplayName) {
				project.CreatorName = userInfo.DisplayName
			}
		}
	}
	//check change owner
	if projectRequest.TargetOwnerId > 0 {
		newOwnerInfo, err := s.GetUserInfo(projectRequest.TargetOwnerId)
		if err != nil {
			return project, err
		}
		project.CreatorId = newOwnerInfo.Id
		project.CreatorName = newOwnerInfo.UserName
		if !utils.IsEmpty(newOwnerInfo.DisplayName) {
			project.CreatorName = newOwnerInfo.DisplayName
		}
	}
	tx := s.db.Begin()

	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProject:save project fail with error: ", err)
		return project, err
	}
	mergeIds := make([]uint64, 0)
	//if is merge project, delete other project
	if !utils.IsEmpty(projectRequest.TargetMergeIds) {
		targetMergeIds := strings.Split(projectRequest.TargetMergeIds, ",")
		for _, targetId := range targetMergeIds {
			if utils.IsEmpty(targetId) {
				continue
			}
			mergeId, err := strconv.ParseInt(targetId, 0, 32)
			if err == nil && mergeId != int64(project.ProjectId) {
				mergeIds = append(mergeIds, uint64(mergeId))
			}
		}
		//delete other merged project
		if len(mergeIds) > 0 {
			for _, id := range mergeIds {
				if id > 0 {
					s.db.Where("project_id = ?", id).Delete(&storage.Project{})
				}
			}
		}
	}

	// update all related data
	payments := make([]*storage.Payment, 0)
	query := fmt.Sprintf(`SELECT * FROM payments WHERE project_id = %d OR details @> '[{"projectId": %d}]'`, projectRequest.ProjectId, projectRequest.ProjectId)
	if err := s.db.Raw(query).Scan(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return project, nil
		}
		return project, err
	}

	// validate payment
	for _, paym := range payments {
		if paym.ProjectId == projectRequest.ProjectId {
			paym.ProjectName = projectRequest.ProjectName
		}
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

	if len(payments) > 0 {
		if err := s.db.Save(&payments).Error; err != nil {
			tx.Rollback()
			log.Error("UpdateProject:update payment info fail with error: ", err)
			return project, err
		}
	}

	tx.Commit()
	//sync payment data
	for _, syncId := range mergeIds {
		s.SyncNewProjectId(syncId, project.ProjectId, project.ProjectName)
	}
	return project, nil
}

func (s *Service) SyncNewProjectId(oldProjectId, newProjectId uint64, newProjectName string) error {
	// update all related data
	payments := make([]*storage.Payment, 0)
	query := fmt.Sprintf(`SELECT * FROM payments WHERE project_id = %d OR details @> '[{"projectId": %d}]'`, oldProjectId, oldProjectId)
	if err := s.db.Raw(query).Scan(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	// validate payment
	for _, paym := range payments {
		if paym.ProjectId == oldProjectId {
			paym.ProjectName = newProjectName
			paym.ProjectId = newProjectId
		}
		if paym.Details != nil {
			details := make([]storage.PaymentDetail, 0)
			for _, detail := range paym.Details {
				if detail.ProjectId == oldProjectId {
					detail.ProjectName = newProjectName
					detail.ProjectId = newProjectId
				}
				details = append(details, detail)
			}
			paym.Details = details
		}
	}
	tx := s.db.Begin()
	if len(payments) > 0 {
		if err := s.db.Save(&payments).Error; err != nil {
			tx.Rollback()
			log.Error("UpdateProject:update payment info fail with error: ", err)
			return err
		}
	}
	tx.Commit()
	return nil
}
