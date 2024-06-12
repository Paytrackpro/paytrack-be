package service

import (
	"sync"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
)

const userFieldLastSeen = "last_seen"

var mutex sync.Mutex

type actionTimeState struct {
	saveUser    map[int]time.Time
	taskRunning bool
}

func NewActionTime() *actionTimeState {
	return &actionTimeState{
		saveUser:    make(map[int]time.Time),
		taskRunning: false,
	}
}

func (s *Service) IsTimeStateRunning() bool {
	return s.timeState.taskRunning
}

func (s *Service) SetLastSeen(id int) {
	mutex.Lock()
	defer mutex.Unlock()
	s.timeState.saveUser[id] = time.Now()
}

func (s *Service) queryUpdateUser(field string, value interface{}, userID int) error {
	var err = s.db.Model(&storage.User{}).Where("id = ?", userID).Update(field, value).Error
	return err
}

func (s *Service) RunTimeTask() {
	// if the task is not running, launch the task
	if !s.timeState.taskRunning {
		s.timeState.taskRunning = true
		go func() {
			// Check last seen status once every 1 minutes
			for range time.Tick(time.Minute) {
				if len(s.timeState.saveUser) > 0 {
					for key, value := range s.timeState.saveUser {
						s.queryUpdateUser(userFieldLastSeen, value, key)
					}
					s.timeState.saveUser = make(map[int]time.Time)
				}
			}
		}()
	}
}

func (s *Service) updateStartDate() error {
	var payments []storage.Payment
	var err = s.db.Where("start_date IS NULL").Find(&payments).Error
	if err == nil {
		for _, payment := range payments {
			var start_date = payment.CreatedAt
			if payment.Details != nil {
				for _, detail := range payment.Details {
					parse_date, err := time.Parse("2006/01/02",detail.Date)
					if err == nil && parse_date.Before(start_date){
						start_date = parse_date
					}
				}
			}
			err = s.db.Model(&storage.Payment{}).Where("id = ?", payment.Id).Update("start_date", start_date).Error
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (s *Service) RunMigrations() error {
	if err := s.updateStartDate(); err != nil {
		return err
	}
	return nil
}
