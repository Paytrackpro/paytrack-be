package webserver

import (
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
)

type lastSeenUser struct {
	userId   int
	lastSeen time.Time
}

type actionTimeState struct {
	saveUser    []lastSeenUser
	taskRunning bool
}

func newTimeState(saveUserArr []lastSeenUser, taskRunningValue bool) *actionTimeState {
	return &actionTimeState{
		saveUser:    saveUserArr,
		taskRunning: taskRunningValue,
	}
}

func newLastSeenUser(userIdVal int, lastSeenVal time.Time) lastSeenUser {
	return lastSeenUser{
		userId:   userIdVal,
		lastSeen: lastSeenVal,
	}
}

func runTimeTask(a *WebServer) {
	//if the task is not running, launch the task
	if !a.timeState.taskRunning {
		a.timeState.taskRunning = true
		go func() {
			//Check last seen status once every 10 minutes
			for range time.Tick(time.Second * 600) {
				if len(a.timeState.saveUser) > 0 {
					for _, state := range a.timeState.saveUser {
						user, err := a.db.QueryUser(storage.UserFieldId, state.userId)
						if err == nil && state.lastSeen.After(user.LastSeen) {
							user.LastSeen = state.lastSeen
							a.db.UpdateUser(user)
						}
					}
					a.timeState.saveUser = make([]lastSeenUser, 0)
				}
			}
		}()
	}
}

/*
	If a user exists in the state, update last seen.

If not already exist, add user to state
*/
func checkAndAddSeenUser(a *WebServer, uID int) {
	var seenUserArr = a.timeState.saveUser
	var arrLen = len(seenUserArr)
	if arrLen > 0 {
		var pop = seenUserArr[arrLen-1]
		if pop.userId == uID {
			a.timeState.saveUser[arrLen-1].lastSeen = time.Now()
			return
		}
	}
	var lastSeenUser = newLastSeenUser(uID, time.Now())
	a.timeState.saveUser = append(a.timeState.saveUser, lastSeenUser)
}
