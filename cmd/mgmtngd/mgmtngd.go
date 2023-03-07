package main

import (
	"encoding/gob"
	"fmt"
	"os"

	"code.cryptopower.dev/mgmt-ng/be/email"
	"code.cryptopower.dev/mgmt-ng/be/log"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

func main() {
	err := _main()
	fmt.Println(err)
}

func _main() error {
	wconfig := &webauthn.Config{
		RPDisplayName: "MGMT",                            // Display Name for your site
		RPID:          "localhost",                       // Generally the FQDN for your site
		RPOrigins:     []string{"http://localhost:8081"}, // The origin URLs allowed for WebAuthn requests
	}

	gob.Register(webauthn.SessionData{})
	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		return err
	}

	// load
	err = godotenv.Load()
	if err != nil {
		return fmt.Errorf("err loading: %v", err)
	}

	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	fmt.Println(os.Getenv("SESSION_KEY"))

	conf, err := loadConfig()
	if err != nil {
		return err
	}
	db, err := storage.NewStorage(conf.Db)
	if err != nil {
		return err
	}

	// Config log
	log.SetLogLevel(conf.LogLevel)
	if err := log.InitLogRotator(conf.LogDir); err != nil {
		return fmt.Errorf("failed to init logRotator: %v", err.Error())
	}

	mailClient, err := email.NewMailClient(conf.Mail)
	if err != nil {
		return err
	}
	web, err := webserver.NewWebServer(conf.WebServer, db, mailClient, webAuthn, store)
	if err != nil {
		return err
	}

	return web.Run()
}
