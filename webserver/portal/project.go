package portal

import "code.cryptopower.dev/mgmt-ng/be/storage"

type ProjectRequest struct {
	ProjectName   string          `json:"projectName"`
	Client        string          `json:"client"`
	Members       storage.Members `json:"members" gorm:"type:jsonb"`
	ProposalToken string          `json:"proposalToken"`
	CreatorId     uint64          `json:"CreatorId"`
}
