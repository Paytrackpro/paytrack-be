package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

type ProjectStatus int

type Project struct {
	ProjectId   uint64        `gorm:"primarykey" json:"projectId"`
	ProjectName string        `json:"projectName"`
	Members     Members       `json:"members" gorm:"type:jsonb"`
	CreatorId   uint64        `json:"creatorId"`
	Status      ProjectStatus `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type Members []Member

type Member struct {
	MemberId    uint64 `json:"memberId"`
	UserName    string `json:"userName"`
	DisplayName string `json:"displayName"`
	Role        int    `json:"role"`
}

// Value Marshal
func (a Members) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *Members) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

const (
	ProjectCreated ProjectStatus = iota
	ProjectConfirmed
	ProjectProcessing
	ProjectCompleted
	ProjectCanceled
	ProjectMaintenance
)

type ProjectFilter struct {
	Id          uint64
	ProjectName string
	CreatorId   uint64
}

func (a ProjectFilter) RequestedSort() string {
	return ""
}
func (a ProjectFilter) BindQuery(db *gorm.DB) *gorm.DB {
	if a.CreatorId > 0 {
		db = db.Where("creator_id = ?", a.CreatorId)
	}
	if !utils.IsEmpty(a.ProjectName) {
		db = db.Where("project_name = ?", a.ProjectName)
	}
	return db
}
func (a ProjectFilter) BindFirst(db *gorm.DB) *gorm.DB {
	if a.CreatorId > 0 {
		db = db.Where("creator_id = ?", a.CreatorId)
	}
	if !utils.IsEmpty(a.ProjectName) {
		db = db.Where("project_name = ?", a.ProjectName)
	}
	return db
}
func (a ProjectFilter) BindCount(db *gorm.DB) *gorm.DB {
	if a.CreatorId > 0 {
		db = db.Where("creator_id = ?", a.CreatorId)
	}
	if !utils.IsEmpty(a.ProjectName) {
		db = db.Where("project_name = ?", a.ProjectName)
	}
	return db
}
func (a ProjectFilter) Sortable() map[string]bool {
	return map[string]bool{}
}
