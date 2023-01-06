package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"net/http"
)

type apiUser struct {
	*WebServer
}

func (a *apiUser) info(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
	} else {
		a.successResponse(w, user)
	}
}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	var updateUser storage.User
	err = a.parseJSON(r, &updateUser)
	if err != nil {
		a.errorResponse(w, err, http.StatusBadRequest)
		return
	}
	user.Email = updateUser.Email
	err = a.db.UpdateUser(user)
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	a.successResponse(w, Map{
		"userId": user.Id,
	})
}
