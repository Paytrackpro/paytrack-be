package storage

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage interface {
	Create(obj interface{}) error
	Save(obj interface{}) error
	GetById(id interface{}, obj interface{}) error
	GetList(f Filter, obj interface{}) error
	First(f Filter, obj interface{}) error
	Count(f Filter, obj interface{}) (int64, error)
	UserStorage
}

type Filter interface {
	Sortable() map[string]bool
	RequestedSort() string
	BindQuery(db *gorm.DB) *gorm.DB
	BindFirst(db *gorm.DB) *gorm.DB
	BindCount(db *gorm.DB) *gorm.DB
}

type psql struct {
	db *gorm.DB
}

type Config struct {
	Dns string `yaml:"dns"`
}

func NewStorage(c Config) (Storage, error) {
	db, err := gorm.Open(postgres.Open(c.Dns), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = autoMigrate(db)
	if err != nil {
		return nil, err
	}
	return &psql{
		db: db,
	}, err
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Payment{}, &PaymentApprover{}, &PaymentApprovalStatus{})
}

func (p *psql) Create(obj interface{}) error {
	return p.db.Create(obj).Error
}
func (p *psql) Save(obj interface{}) error {
	return p.db.Save(obj).Error
}

func (p *psql) GetById(id interface{}, obj interface{}) error {
	return p.db.Where("id = ?", id).First(obj).Error
}

func (p *psql) GetList(f Filter, obj interface{}) error {
	err := f.BindQuery(p.db).Find(obj).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	return err
}

func (p *psql) First(f Filter, obj interface{}) error {
	return f.BindFirst(p.db).Find(obj).Error
}

func (p *psql) Count(f Filter, obj interface{}) (int64, error) {
	var count int64
	var err = f.BindCount(p.db.Model(obj)).Count(&count).Error
	return count, err
}
