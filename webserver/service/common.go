package service

import (
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
)

const UserFieldLastSeen = "last_seen"

type ActionTimeState struct {
	SaveUser    map[int]time.Time
	TaskRunning bool
}

func (s *Service) QueryUpdateUser(field string, value interface{}, userID int) error {
	var err = s.db.Model(&storage.User{}).Where("id = ?", userID).Update(field, value).Error
	return err
}

func (s *Service) RunTimeTask() {
	//if the task is not running, launch the task
	if !s.TimeState.TaskRunning {
		s.TimeState.TaskRunning = true
		go func() {
			//Check last seen status once every 5 minutes
			for range time.Tick(time.Minute * 5) {
				if len(s.TimeState.SaveUser) > 0 {
					for key, value := range s.TimeState.SaveUser {
						s.QueryUpdateUser(UserFieldLastSeen, value, key)
					}
					s.TimeState.SaveUser = make(map[int]time.Time)
				}
			}
		}()
	}
}
