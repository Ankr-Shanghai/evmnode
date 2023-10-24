# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_COMMIT_DATE=$(shell git log -n1 --pretty='format:%cd' --date=format:'%Y%m%d')

geth:
	go build -o ./bin/geth -ldflags "-X main.gitCommit=$(GIT_COMMIT) -X main.gitCommitDate=$(GIT_COMMIT_DATE)" \
	./cmd/geth
	@echo "Done building."
