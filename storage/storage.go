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
	UserStorage
}

type Filter interface {
	Sortable() map[string]bool
	RequestedSort() string
	BindQuery(db *gorm.DB) *gorm.DB
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
	return db.AutoMigrate(&User{}, &Payment{})
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
