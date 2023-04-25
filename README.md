# [Application Scope](https://code.cryptopower.dev/mgmt-ng/fe/-/wikis/home)

# MGMT-NG backend

mgmt-ng backend(mgmtngd) is the code of backend for mgmt-ng project. It is written in [go](https://golang.org/) and using go 1.19.

## Setup

### Database ([Posgresql](https://www.postgresql.org/))

When you have Posgresql, you need create new db with name **mgmtng**

### Config env

Create new yaml config on ./private/config.yaml

This is content of config.yaml:

```
db:
  dns: "host=<host> user=<user> password=<password> dbname=mgmtng port=<port> sslmode=disable TimeZone=Asia/Shanghai"
webServer:
  # port: the port mgmtngd will take to run the web server
  port: <port>

  # hmacSecretKey: used to generate jwt hash. it should be private on production
  hmacSecretKey: <key>

  # aliveSessionHours: time to keep the login session alive
  aliveSessionHours: 3000

  # aesSecretKey: a secret key used to encrypt sensitive data
  aesSecretKey: asder45t678uio89

  service:
    # config to CEX use to convert coin rate
    # we support 3 CEXs: "binance", "bittrex", "coinmarketcap"
    # "coinmarketcap" requires API key
    exchange: "coinmarketcap"

    # API key needed when use coinmarketcap
    coimarketcapKey: <API key>

# Config log level: "trace", "debug", "info", "warn", "error", "off"
logLevel: <log level>

# The path where mgmt.log will be saved exp: ./root/mgmtlog
logDir: <Path>

mail:
  # the address of mail server exp: smtp.gmail.com:587
  addr: <address>
  # the username of mail server
  userName: your_user_name@gmail.com
  # password: taken from: https://myaccount.google.com/security
  # click on 'App passwords'
  password: your_password
  host: smtp.gmail.com
  # from: send mail from. the same with userName for google service
  from: mail_from@example.com

```

## Running mgmtngd (Linux | MacOS | Window):

### Terminal

Run `go run ./cmd/mgmtngd --config=./private/config.yaml`

### Makefile

You can create `Makefile` and add command to makefile like this

```
.PHONY:
up:
	go run ./cmd/mgmtngd --config=./private/config.yaml

```

after that run `make up`
