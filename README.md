# [Application Scope](https://code.cryptopower.dev/mgmt-ng/fe/-/wikis/home)

# MGMT-NG backend

mgmt-ng backend(mgmtngd) is the code of backend for mgmt-ng project. It is written in [go](https://golang.org/) and using  go 1.19.

mgmtngd is using [postgresql](https://www.postgresql.org/) to be database. So for running mgmtngd, please get postgresql up and running.

Running mgmtngd (Linux or MacOS): 
```
go run ./cmd/mgmtngd/ -config=path_to_config_file.yaml
```
The sample config file can be found at `./sample/mgmtngd.yaml`. For development, you should create a folder `./private` and copy the file into this place.
In this project, the `private` folder is ignored by git and will not affect if you change the value for the configuration.
