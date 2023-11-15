package service

import (
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
)

func (s *Service) CreateNewProject(userId uint64, projectRequest portal.ProjectRequest) (*storage.Project, error) {
	newProject := storage.Project{
		ProjectName:   projectRequest.ProjectName,
		Client:        projectRequest.Client,
		Members:       projectRequest.Members,
		ProposalToken: projectRequest.ProposalToken,
		CreatorId:     userId,
		Status:        storage.ProjectConfirmed,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
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
