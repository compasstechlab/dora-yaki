ENV_FILE ?= ../.env

ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif

project ?= $(GCP_PROJECT_ID)
region ?= $(GCP_REGION)
backend_function_name ?= $(BACKEND_FUNCTION_NAME)
backend_service_account_id ?= $(BACKEND_SERVICE_ACCOUNT_ID)
oauth_redirect_url ?= $(BACKEND_OAUTH_REDIRECT_URL)
frontend_url ?= $(BACKEND_FRONTEND_URL)
backend_encryption_kms_key ?= $(BACKEND_ENCRYPTION_KMS_KEY)

ifneq ($(strip $(backend_encryption_kms_key)),)
encryption_deploy_flag := --set-env-vars="ENCRYPTION_KMS_KEY=$(backend_encryption_kms_key)"
encryption_summary := BACKEND_ENCRYPTION_KMS_KEY=$(backend_encryption_kms_key) -> ENCRYPTION_KMS_KEY
else
encryption_deploy_flag := --set-secrets=ENCRYPTION_KEY_BASE64=ENCRYPTION_KEY_BASE64:latest
encryption_summary := ENCRYPTION_KEY_BASE64=(Secret Manager: ENCRYPTION_KEY_BASE64)
endif

init:
	go mod download

build:
	go build -o ./bin/ ./cmd/httpserver

dev:
	go run ./cmd/httpserver/main.go

lint:
	golangci-lint run ./...

lint-fix:
	go mod tidy
	golangci-lint run ./... --fix


deploy:
	$(MAKE) __validate-deploy
	$(MAKE) __confirm-deploy
	gcloud functions deploy $(backend_function_name) \
	  --gen2 \
	  --project $(project) \
	  --region $(region) \
	  --runtime go125 \
	  --source ./ \
	  --entry-point RunHTTPServer \
	  --trigger-http \
	  --timeout=540s \
	  --set-env-vars="ENVIRONMENT=production" \
	  --set-env-vars="TZ_OFFSET=$(TZ_OFFSET)" \
	  --set-env-vars="OAUTH_REDIRECT_URL=$(oauth_redirect_url)" \
	  --set-env-vars="FRONTEND_URL=$(frontend_url)" \
	  --set-secrets=GITHUB_OAUTH_CLIENT_ID=GITHUB_OAUTH_CLIENT_ID:latest \
	  --set-secrets=GITHUB_OAUTH_CLIENT_SECRET=GITHUB_OAUTH_CLIENT_SECRET:latest \
	  --set-secrets=AUTH_JWT_SECRET=AUTH_JWT_SECRET:latest \
	  $(encryption_deploy_flag) \
	  --set-secrets=JOB_AUTH_KEY=JOB_AUTH_KEY:latest \
	  --service-account=$(backend_service_account_id)@$(project).iam.gserviceaccount.com \
	  --allow-unauthenticated

__validate-deploy: __validate
	@if [ "$(backend_function_name)" = "" ]; then echo "BACKEND_FUNCTION_NAME is not set"; exit 1; fi
	@if [ "$(backend_service_account_id)" = "" ]; then echo "BACKEND_SERVICE_ACCOUNT_ID is not set"; exit 1; fi
	@if [ "$(oauth_redirect_url)" = "" ]; then echo "OAUTH_REDIRECT_URL is not set"; exit 1; fi
	@if [ "$(frontend_url)" = "" ]; then echo "FRONTEND_URL is not set"; exit 1; fi

__confirm-deploy:
	@echo "Deploy backend with these variables:"
	@echo "  ENV_FILE=$(ENV_FILE)"
	@echo "  GCP_PROJECT_ID=$(project)"
	@echo "  GCP_REGION=$(region)"
	@echo "  BACKEND_FUNCTION_NAME=$(backend_function_name)"
	@echo "  RUNTIME=go125"
	@echo "  ENTRY_POINT=RunHTTPServer"
	@echo "  TIMEOUT=540s"
	@echo "  ENVIRONMENT=production"
	@echo "  BACKEND_SERVICE_ACCOUNT_ID=$(backend_service_account_id)"
	@echo "  TZ_OFFSET=$(TZ_OFFSET)"
	@echo "  OAUTH_REDIRECT_URL=$(oauth_redirect_url)"
	@echo "  FRONTEND_URL=$(frontend_url)"
	@echo "  $(encryption_summary)"
	@echo "  JOB_AUTH_KEY=(Secret Manager: JOB_AUTH_KEY)"
	@printf "Continue? [y/N] "; \
	read ans; \
	case "$$ans" in y|Y|yes|YES) ;; *) echo "Deploy cancelled."; exit 1;; esac

__validate:
	@if [ "$(project)" = "" ]; then echo "project is not set"; exit 1; fi
	@if [ "$(region)" = "" ]; then echo "region is not set"; exit 1; fi
