package portal

import "code.cryptopower.dev/mgmt-ng/be/storage"

type ProjectRequest struct {
	ProjectId   uint64          `json:"projectId"`
	ProjectName string          `json:"projectName"`
	Members     storage.Members `json:"members" gorm:"type:jsonb"`
	CreatorId   uint64          `json:"CreatorId"`
}
