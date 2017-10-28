DOCKER_ARGS=
APP=ap-5r
PROJECT=swgoh-api
TOKEN=$(shell cat .token 2>/dev/null)

build:
	go build -o $(APP)
	docker build \
		-t gcr.io/ronoaldoconsulting/$(APP):latest \
		--build-arg GIT_HASH=$$(git rev-parse --short HEAD) .

run: build
	docker run --name ap-5r \
		--rm \
		--env USE_DEV=true \
		--env BOT_TOKEN=$(TOKEN) \
	       	-it $(DOCKER_ARGS) \
		gcr.io/ronoaldoconsulting/$(APP):latest

deploy: build
	gcloud --project=$(PROJECT) docker -- \
		push gcr.io/ronoaldoconsulting/$(APP):latest
	gcloud --project=$(PROJECT) compute \
		ssh chatbots < scripts/reload.sh

logs:
	gcloud --project=$(PROJECT) compute \
		ssh chatbots < scripts/keep-logging.sh
