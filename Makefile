include .env

test:
	go test

integration-test:
	go test -run-integration-tests
