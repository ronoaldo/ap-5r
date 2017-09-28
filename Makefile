build:
	go build -o ap-5r
	docker build -t gcr.io/ronoaldoconsulting/ap-5r:latest --build-arg GIT_HASH=$$(git rev-parse --short HEAD) .

run: build
	docker run --rm --env USE_DEV=true -it gcr.io/ronoaldoconsulting/ap-5r:latest

deploy: build
	gcloud docker -- push gcr.io/ronoaldoconsulting/ap-5r:latest
	gcloud compute ssh --command="/bin/bash /home/ronoaldo/reload.sh" chatbot
