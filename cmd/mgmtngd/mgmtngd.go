package main

import (
	"fmt"

	"github.com/Paytrackpro/paytrack-be/email"
	"github.com/Paytrackpro/paytrack-be/log"
	"github.com/Paytrackpro/paytrack-be/storage"
	"github.com/Paytrackpro/paytrack-be/webserver"
)

func main() {
	err := _main()
	fmt.Println(err)
}

func _main() error {
	conf, err := loadConfig()
	if err != nil {
		return err
	}

	// Config log
	log.SetLogLevel(conf.LogLevel)
	if err := log.InitLogRotator(conf.LogDir); err != nil {
		return fmt.Errorf("failed to init logRotator: %v", err.Error())
	}

	db, err := storage.NewStorage(conf.Db, log.GetDBLogger())
	if err != nil {
		return err
	}

	mailClient, err := email.NewMailClient(conf.Mail)
	if err != nil {
		return err
	}
	web, err := webserver.NewWebServer(conf.WebServer, db, mailClient)
	if err != nil {
		return err
	}

	return web.Run()
}
