#############################################################################################################

.PHONY: all clean complainer complainer.linux complainer.mac docker docker-dev upload-docker print-docker-repo-tag require-docker-account pre-commit vet lint fmt fmt-list fmt-diff fmt-check ineffassign

VERSION ?= dev
DOCKER_IMAGE_NAME ?= complainer
DOCKER_ACCOUNT ?= $(shell if [ -e env.sh ]; then . env.sh ; echo $$_DOCKER_ACCOUNT ; fi)

ifeq ($(DOCKER_ACCOUNT), "")
DOCKER_REPO_TAG=$(DOCKER_IMAGE_NAME):$(VERSION)
else
DOCKER_REPO_TAG=$(DOCKER_ACCOUNT)/$(DOCKER_IMAGE_NAME):$(VERSION)
endif

UNIT_TEST_TAGS="unit"
INTEGRATION_TEST_TAGS="integration"

SOURCE=cmd/complainer/main.go
TARGET=complainer

GOFMT_OPTIONS=-s
PACKAGES=$(shell go list ./... | grep -vE '^github.com/cloudflare/complainer/vendor/')
CHECK_FILES=$(shell find . -type f -name '*.go' | grep -vE '^\./vendor/')
UNIT_TEST_PKGS=$(shell go list ./... | grep -v "/vendor/" )

#############################################################################################################

all: clean test $(TARGET) docker

clean:
	if [ -e "$(TARGET)" ]; then rm -v $(TARGET) ; fi
	if [ -e "$(TARGET).linux" ]; then rm -v $(TARGET).linux ; fi
	if [ -e "$(TARGET).mac" ]; then rm -v $(TARGET).mac ; fi

	if [ $$(docker images $(DOCKER_REPO_TAG) -q | wc -l) == 1 ]; then \
		echo docker rmi -f $(DOCKER_REPO_TAG); \
		docker rmi -f $(DOCKER_REPO_TAG); \
	fi

$(TARGET):
	go build -o $(TARGET) $(SOURCE)

$(TARGET).linux:
	GOOS=linux GOARCH=amd64 go build -o $(TARGET).linux $(SOURCE)

$(TARGET).mac:
	GOOS=darwin GOARCH=amd64 go build -o $(TARGET).mac $(SOURCE)

docker:
	docker build -f Dockerfile -t $(DOCKER_REPO_TAG) .

docker-dev:
	docker build -f Dockerfile.dev -t $(DOCKER_REPO_TAG) .

# Only upload if the image name is "<accountname>/<imagename>:<tag>"
upload-docker:
	@if [ -z $(DOCKER_ACCOUNT) ]; then echo ERROR: Variable DOCKER_ACCOUNT missing ; exit 1; fi
	docker push $(DOCKER_REPO_TAG)

print-docker-repo-tag:
	@echo $(DOCKER_REPO_TAG)

#############################################################################################################
# Tests

test: vet fmt-check lint unit ineffassign

unit:
	go test -v -cover -timeout 10s -race $(UNIT_TEST_PKGS)

pre-commit: vet lint fmt ineffassign

vet:
	@echo "Running go vet"
	@go vet $(CHECK_PKGS)

lint:
	golint -set_exit_status $(PACKAGES)

fmt:
	go fmt $(PACKAGES)

fmt-list:
	@gofmt $(GOFMT_OPTIONS) -l $(CHECK_FILES)

fmt-diff:
	@gofmt $(GOFMT_OPTIONS) -d $(CHECK_FILES)

fmt-check:
ifneq ($(shell gofmt $(GOFMT_OPTIONS) -l $(CHECK_FILES) | wc -l | tr -d "[:blank:]"), 0)
	$(error gofmt returns more than one line, run 'make fmt-check' or 'make fmt-diff' for details, 'make fmt' to fix)
endif
	@echo "gofmt check successful"

#go get -u github.com/gordonklaus/ineffassign
ineffassign:
	ineffassign ./
