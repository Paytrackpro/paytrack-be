package portal

import "code.cryptopower.dev/mgmt-ng/be/storage"

type ProjectRequest struct {
	ProjectId      uint64          `json:"projectId"`
	ProjectName    string          `json:"projectName"`
	Members        storage.Members `json:"members" gorm:"type:jsonb"`
	Approvers      storage.Members `json:"approvers" gorm:"type:jsonb"`
	Description    string          `json:"description"`
	CreatorId      uint64          `json:"CreatorId"`
	TargetOwnerId  uint64          `json:"targetOwnerId"`
	TargetMergeIds string          `json:"targetMergeIds"`
}
