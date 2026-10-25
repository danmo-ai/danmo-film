APP_NAME := danmo-film
FRONTEND_DIR := $(CURDIR)/frontend
OUT_DIR := $(CURDIR)/out

export DQ_APP_NAME := $(APP_NAME)
DQ_BACKEND_PORT ?= 7803
DQ_FRONTEND_PORT ?= 5803
export DQ_BACKEND_PORT
export DQ_FRONTEND_PORT

.PHONY: help backend dev start stop \
	frontend-install frontend-dev frontend-build \
	test health clean

help:
	@echo "Danmo Film"
	@echo ""
	@echo "Dev:"
	@echo "  dev | start   Backend + Vite (browser :$(DQ_FRONTEND_PORT) / API :$(DQ_BACKEND_PORT))"
	@echo "  backend       Start backend only (:$(DQ_BACKEND_PORT))"
	@echo "  stop          Stop all dev processes"
	@echo ""
	@echo "Frontend:  frontend-install | frontend-dev | frontend-build"
	@echo "Test:      test | health"
	@echo "Clean:     clean"

dev start:
	@chmod +x scripts/*.sh
	@./scripts/start.sh

backend:
	@chmod +x scripts/*.sh
	@./scripts/start_backend.sh

stop:
	@chmod +x scripts/*.sh
	@./scripts/stop.sh

frontend-install:
	cd $(FRONTEND_DIR) && npm install

frontend-dev: frontend-install
	cd $(FRONTEND_DIR) && DQ_FRONTEND_PORT=$(DQ_FRONTEND_PORT) DQ_BACKEND_PORT=$(DQ_BACKEND_PORT) npm run dev

frontend-build: frontend-install
	cd $(FRONTEND_DIR) && npm run build

test:
	go test ./...

health:
	curl -sS "http://127.0.0.1:$(DQ_BACKEND_PORT)/api/system/health"

clean:
	rm -rf $(OUT_DIR)
