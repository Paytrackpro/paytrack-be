package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"net/http"
)

type apiUser struct {
	*WebServer
}

type userForm struct {
	Email string `validate:"required,email"`
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
	var f userForm
	err := a.parseJSON(r, &f)
	if err != nil {
		a.errorResponse(w, err, http.StatusBadRequest)
		return
	}
	err = a.validator.Struct(&f)
	if err != nil {
		a.errorResponse(w, err, http.StatusBadRequest)
		return
	}
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	user.Email = f.Email
	err = a.db.UpdateUser(user)
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	a.successResponse(w, Map{
		"userId": user.Id,
	})
}
