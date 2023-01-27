package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type apiAuth struct {
	*WebServer
}

type authForm struct {
	UserName string `validate:"required,alphanum,gte=4,lte=32"`
	Password string `validate:"required"`
}

type authClaims struct {
	Id     uint64
	Expire int64
}

const authClaimsCtxKey = "authClaimsCtxKey"

func (c authClaims) Valid() error {
	timestamp := time.Now().Unix()
	if timestamp >= c.Expire {
		return fmt.Errorf("the credential is expired")
	}
	return nil
}

func (a *apiAuth) register(w http.ResponseWriter, r *http.Request) {
	var f authForm
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
	hash, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	var user = storage.User{
		UserName:     f.UserName,
		PasswordHash: string(hash),
	}
	if err := a.db.CreateUser(&user); err != nil {
		// 23505 is  duplicate key value error for postgresql
		if e, ok := err.(*pgconn.PgError); ok && e.Code == "23505" && e.ConstraintName == "users_user_name_idx" {
			mess := fmt.Sprintf("the user name '%s' is already taken", f.UserName)
			er := utils.NewError(mess, utils.ErrorObjectExist)
			utils.Response(w, http.StatusBadRequest, er, nil)
			return
		}
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, nil, Map{
		"userId": user.Id,
	})
}

func (a *apiAuth) login(w http.ResponseWriter, r *http.Request) {
	var f authForm
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	user, err := a.db.QueryUser(storage.UserFieldUName, f.UserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.LoginFail, nil)
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
		}
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.LoginFail, nil)
		return
	}
	var authClaim = authClaims{
		Id:     user.Id,
		Expire: time.Now().Add(time.Hour * time.Duration(a.conf.AliveSessionHours)).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
	tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, nil, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}
