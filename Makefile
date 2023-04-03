##### Local run #####
.PHONY:
up:
	go run ./cmd/mgmtngd --config=./private/config.yaml
