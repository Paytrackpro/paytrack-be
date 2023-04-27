# [Application Scope](https://code.cryptopower.dev/mgmt-ng/fe/-/wikis/home)

# MGMT-NG backend

mgmt-ng backend(mgmtngd) is the code of backend for mgmt-ng project. It is written in [go](https://golang.org/) and using go 1.19.

## Setup

### Database ([Posgresql](https://www.postgresql.org/))

When you have Posgresql, you need create new db with name **mgmtng**

### Config env

Create new yaml config on ./private/config.yaml

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
