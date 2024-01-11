## ----------------------------------------------------------------------------
## Make targets to test deployments locally.
## ----------------------------------------------------------------------------

.PHONY: build
## build : build docker image for hugo application
build:
	docker build . -t hungrymouse

.PHONY: deploy
## deploy : deploy hugo application in docker container
deploy:
	docker run --rm -p 8080:8080 hungrymouse