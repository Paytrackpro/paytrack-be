package main

import (
	"fmt"

	"code.cryptopower.dev/mgmt-ng/be/email"
	"code.cryptopower.dev/mgmt-ng/be/log"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver"
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
	web, err := webserver.NewWebServer(conf.WebServer, db, mailClient)
	if err != nil {
		return err
	}

	return web.Run()
}
