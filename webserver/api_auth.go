package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type apiAuth struct {
	*WebServer
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
	var f portal.RegisterForm
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
	user, err := f.User()
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	if err := a.db.CreateUser(user); err != nil {
		// 23505 is  duplicate key value error for postgresql
		if e, ok := err.(*pgconn.PgError); ok && e.Code == utils.PgsqlDuplicateErrorCode {
			if e.ConstraintName == "users_user_name_idx" {
				a.errorResponse(w, fmt.Errorf("the user name '%s' is already taken", f.UserName), http.StatusBadRequest)
				return
			}
		}
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	a.successResponse(w, Map{
		"userId": user.Id,
	})
}

func (a *apiAuth) login(w http.ResponseWriter, r *http.Request) {
	var f portal.LoginForm
	err := a.parseJSON(r, &f)
	if err != nil {
		a.errorResponse(w, err, http.StatusBadRequest)
		return
	}
	user, err := a.db.QueryUser(storage.UserFieldUName, f.UserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a.errorResponse(w, fmt.Errorf("your user name or password is incorrect"), http.StatusBadRequest)
		} else {
			a.errorResponse(w, err, http.StatusInternalServerError)
		}
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		a.errorResponse(w, fmt.Errorf("your user name or password is incorrect"), http.StatusBadRequest)
		return
	}
	var authClaim = authClaims{
		Id:     user.Id,
		Expire: time.Now().Add(time.Hour * time.Duration(a.conf.AliveSessionHours)).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
	tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
	if err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	a.successResponse(w, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}
