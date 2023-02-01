package webserver

import (
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/log"
	"code.cryptopower.dev/mgmt-ng/be/models"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
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
		utils.ResponseOK(w, nil, user)
	}
}

func (a *apiUser) infoWithId(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := a.db.QueryUser(storage.UserFieldId, id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, nil, user)
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
	utils.ResponseOK(w, nil, Map{
		"user_id": user.Id,
	})
}

func (a *apiUser) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	var f portal.UpdateUserRequest
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	err = a.validator.Struct(&f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	a.updateUser(w, f)
}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	var f portal.UpdateUserRequest
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	f.UserId = int(claims.Id)

	err = a.validator.Struct(&f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	a.updateUser(w, f)
}

func (a *apiUser) getListUsers(w http.ResponseWriter, r *http.Request) {
	var query portal.ListUserRequest
	if err := utils.DecodeQuery(&query, r.URL.Query()); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	filter := models.UserFilter{
		KeySearch: query.KeySearch,
		MSort: models.MSort{
			SortType: query.SortType,
			Sort:     query.Sort,
			Limit:    query.Limit,
			Offset:   query.Offset,
		},
	}
	users, err := a.db.GetListUser(filter)
	if err != nil {
		log.Logger.Error(err)
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, nil, users)
}
