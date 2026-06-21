.PHONY: help build build-backend build-frontend

.DEFAULT_GOAL := help
DOCKER_USER ?= octolink

help:
	@echo "Usage: make [build|build-backend|build-frontend] [DOCKER_USER=<user>]"

build: build-frontend build-backend

build-backend:
	@make build -C ../backend/services/controller/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/acs/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/utils/socketio/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/utils/file-server/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/adapter/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/ws-adapter/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/ws/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/mqtt-adapter/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/mqtt/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/stomp-adapter/build/ DOCKER_USER=${DOCKER_USER}
	@make build -C ../backend/services/mtp/stomp/build/ DOCKER_USER=${DOCKER_USER}

build-frontend:
	@make build -C ../frontend/build/ DOCKER_USER=${DOCKER_USER}
