package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type apiAuth struct {
	*WebServer
}

type authForm struct {
	UserName string
	Password string
}

type authClaims struct {
	Id       string
	UserName string
	Expire   int64
}

const authClaimsCtxKey = "authClaimsCtxKey"

const hmacDefaultSecret = "secret"

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
		a.errorResponse(w, err, http.StatusBadRequest)
	} else {
		// todo: validate the input data
		// UserName has at least 4 chars no special chars and no more than 32 chars.
		hash, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)
		if err != nil {
			a.errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		var user = storage.User{
			Id:           "",
			UserName:     f.UserName,
			PasswordHash: string(hash),
			Email:        "",
		}
		err = a.db.CreateUser(&user)
		if err == nil {
			a.successResponse(w, Map{
				"userId": user.Id,
			})
		} else {
			a.errorResponse(w, err, http.StatusInternalServerError)
		}
	}
}

func (a *apiAuth) login(w http.ResponseWriter, r *http.Request) {
	var f authForm
	err := a.parseJSON(r, &f)
	if err != nil {
		a.errorResponse(w, err, http.StatusBadRequest)
	} else {
		var user, err = a.db.QueryUser(storage.UserFieldUName, f.UserName)
		if err != nil {
			a.errorResponse(w, fmt.Errorf("your user name or password is incorrect"), http.StatusBadRequest)
			return
		}
		var authClaim = authClaims{
			Id:       user.Id,
			UserName: user.UserName,
			Expire:   time.Now().Add(time.Hour * 24).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
		tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
		if err != nil {
			a.errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		a.successResponse(w, Map{
			"requestToken": tokenString,
			"userInfo":     authClaim,
		})
	}
}
