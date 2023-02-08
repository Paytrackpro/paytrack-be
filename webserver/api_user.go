package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"errors"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
)

type apiUser struct {
	*WebServer
}

func (a *apiUser) info(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, user)
	}
}

func (a *apiUser) infoWithId(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := a.db.QueryUser(storage.UserFieldId, id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, user)
	}
}

// This function use for update user for admin and user
func (a *apiUser) updateUser(w http.ResponseWriter, req portal.UpdateUserRequest) {

	user, err := a.db.QueryUser(storage.UserFieldId, req.UserId)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	utils.SetValue(&user.Email, req.Email)
	utils.SetValue(&user.PaymentType, req.PaymentType)
	utils.SetValue(&user.PaymentAddress, req.PaymentAddress)
	if !utils.IsEmpty(req.Password) {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
		user.PasswordHash = string(hash)
	}
	err = a.db.UpdateUser(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, Map{
		"userId": user.Id,
	})
}

func (a *apiUser) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	var f portal.UpdateUserRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	a.updateUser(w, f)
}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	var f portal.UpdateUserRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	f.UserId = int(claims.Id)
	a.updateUser(w, f)
}

func (a *apiUser) getListUsers(w http.ResponseWriter, r *http.Request) {
	var f storage.UserFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var users []storage.User
	if err := a.db.GetList(&f, &users); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	utils.ResponseOK(w, users)
}

func (a *apiUser) checkingUserExist(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userName")
	user, err := a.db.QueryUser(storage.UserFieldUName, userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ResponseOK(w, Map{
				"found": false,
			})
		} else {
			utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		}
		return
	}
	utils.ResponseOK(w, Map{
		"found":    true,
		"id":       user.Id,
		"userName": user.UserName,
	})
}
