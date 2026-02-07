init:
	go mod download

build:
	go build -o ./bin/ ./cmd/httpserver

dev:
	FUNCTION_TARGET=RunHTTPServer \
	TZ_OFFSET=+09:00 \
	go run ./cmd/httpserver/main.go

lint:
	golangci-lint run ./...

lint-fix:
	go mod tidy
	golangci-lint run ./... --fix


deploy:
	$(MAKE) __validate
	gcloud functions deploy dora-yaki-backend \
	  --gen2 \
	  --project $(project) \
	  --region $(region) \
	  --runtime go125 \
	  --source ./ \
	  --entry-point RunHTTPServer \
	  --trigger-http \
	  --timeout=540s \
	  --set-env-vars="ENVIRONMENT=production" \
	  --set-env-vars="TZ_OFFSET=+09:00" \
	  --update-secrets=GITHUB_TOKEN=GITHUB_TOKEN:latest \
	  --service-account=dora-yaki-api@$(project).iam.gserviceaccount.com \
	  --allow-unauthenticated

schedule-create:
	$(MAKE) __schedule action=create param='{"interval": 60}'

schedule-update:
	$(MAKE) __schedule action=update param='{"interval": 60}'

__schedule:
	$(MAKE) __validate
	$(eval headers_flag := $(if $(filter create,$(action)),--headers,--update-headers))
	gcloud scheduler jobs $(action) http dora-yaki-backend-job-sync \
	  --schedule="*/5 * * * *" \
	  --project=$(project) \
	  --attempt-deadline=540s \
	  --oidc-service-account-email=dora-yaki-api@$(project).iam.gserviceaccount.com \
	  --http-method=PUT \
	  --uri="https://$(region)-$(project).cloudfunctions.net/dora-yaki-backend/api/job/sync" \
	  $(headers_flag)="Content-Type=application/json" \
	  --message-body='$(param)' \
	  --location=$(region)

__validate:
	@if [ "$(project)" = "" ]; then echo "project is not set"; exit 1; fi
	@if [ "$(region)" = "" ]; then echo "region is not set"; exit 1; fi
