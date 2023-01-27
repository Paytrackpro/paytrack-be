package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
)

type Config struct {
	Port              int    `yaml:"port"`
	HmacSecretKey     string `yaml:"hmacSecretKey"`
	AliveSessionHours int    `yaml:"aliveSessionHours"`
}

type WebServer struct {
	mux       *chi.Mux
	conf      *Config
	db        storage.Storage
	validator *validator.Validate
}

type Map map[string]interface{}

func NewWebServer(c Config, db storage.Storage) (*WebServer, error) {
	if c.Port == 0 {
		return nil, fmt.Errorf("please set up server port")
	}
	if c.AliveSessionHours <= 0 {
		return nil, fmt.Errorf("aliveSessionHours must be > 0")
	}
	if c.HmacSecretKey == "" {
		return nil, fmt.Errorf("please set up hmacSecretKey")
	}

	return &WebServer{
		mux:       chi.NewRouter(),
		conf:      &c,
		db:        db,
		validator: validator.New(),
	}, nil
}

func (s *WebServer) Run() error {
	s.Route()
	log.Printf("mgmtngd is running on :%d", s.conf.Port)
	var server = http.Server{
		Addr:              fmt.Sprintf(":%d", s.conf.Port),
		Handler:           s.mux,
		TLSConfig:         nil,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
		MaxHeaderBytes:    0,
		TLSNextProto:      nil,
		ConnState:         nil,
		ErrorLog:          nil,
		BaseContext:       nil,
		ConnContext:       nil,
	}
	return server.ListenAndServe()
}

func (s *WebServer) parseJSON(r *http.Request, data interface{}) error {
	if r.Body == nil {
		return utils.NewError("body cannot be empty or nil", utils.ErrorBodyRequited)
	}
	var decoder = json.NewDecoder(r.Body)
	var err = decoder.Decode(data)
	defer r.Body.Close()
	return err
}

func (s *WebServer) loggedInMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var bearer = r.Header.Get("Authorization")
		// Should be a bearer token
		if len(bearer) > 6 && strings.ToUpper(bearer[0:7]) == "BEARER " {
			var tokenStr = bearer[7:]
			var claim authClaims
			token, err := jwt.ParseWithClaims(tokenStr, &claim, func(token *jwt.Token) (interface{}, error) {
				return []byte(s.conf.HmacSecretKey), nil
			})
			if err != nil {
				utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
				return
			}
			ctx := context.WithValue(r.Context(), authClaimsCtxKey, token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
	}
	return http.HandlerFunc(fn)
}

func (s *WebServer) credentialsInfo(r *http.Request) (*authClaims, bool) {
	val := r.Context().Value(authClaimsCtxKey)
	claims, ok := val.(*authClaims)
	return claims, ok
}
