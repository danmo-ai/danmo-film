.PHONY: dev backend frontend install install-frontend health

DQ_BACKEND_PORT ?= 7803
DQ_FRONTEND_PORT ?= 5803

install:
	python3.11 -m venv .venv
	. .venv/bin/activate && pip install -r requirements.txt

install-frontend:
	cd frontend && npm install

backend:
	DQ_BACKEND_PORT=$(DQ_BACKEND_PORT) FILM_RELOAD=1 .venv/bin/python -m backend.main

frontend:
	cd frontend && DQ_FRONTEND_PORT=$(DQ_FRONTEND_PORT) DQ_BACKEND_PORT=$(DQ_BACKEND_PORT) npm run dev

dev:
	@echo "Run Make API on :7800, then: make backend & make frontend"
	@echo "Film UI http://127.0.0.1:$(DQ_FRONTEND_PORT)  API http://127.0.0.1:$(DQ_BACKEND_PORT)"

health:
	curl -sS "http://127.0.0.1:$(DQ_BACKEND_PORT)/api/system/health" | python3 -m json.tool
