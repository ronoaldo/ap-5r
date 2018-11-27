DOCKER_ARGS=
APP=ap-5r
PROJECT=swgoh-api
TOKEN=$(shell cat .token 2>/dev/null)

build:
	rm -f ap-5r && GOARCH=amd64 GOOS=linux go build -o $(APP)
	docker build \
		-t ronoaldo/$(APP):latest \
		--build-arg GIT_HASH=$$(git rev-parse --short HEAD) .

run: build
	docker run --name ap-5r \
		--rm \
		--link pagerender \
		--env USE_DEV=true \
		--env BOT_TOKEN=$(TOKEN) \
		--env API_USERNAME=$(SWGOH_API_USERNAME) \
		--env API_PASSWORD=$(SWGOH_API_PASSWORD) \
		-it $(DOCKER_ARGS) \
		ronoaldo/$(APP):latest

deploy: build
	docker push ronoaldo/$(APP):latest

gce-reload:
	gcloud --project=$(PROJECT) compute \
		ssh chatbots < scripts/reload.sh

gce-logs:
	gcloud --project=$(PROJECT) compute \
		ssh chatbots < scripts/keep-logging.sh
