package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
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
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, nil, user)
	}
}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {
	var f userForm
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
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	user.Email = f.Email
	err = a.db.UpdateUser(user)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, nil, Map{
		"userId": user.Id,
	})
}

func (a *apiUser) getListUsers(w http.ResponseWriter, r *http.Request) {
	var query portal.ListUserRequest
	if err := utils.DecodeQuery(&query, r.URL.Query()); err != nil {
		fmt.Println("---errr--->", err)
		return
	}
	data, _ := json.Marshal(query)
	fmt.Println(string(data))
}
