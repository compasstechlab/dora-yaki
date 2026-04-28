ENV_FILE ?= ../.env

ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif

project ?= $(GCP_PROJECT_ID)
region ?= $(GCP_REGION)
lang ?= $(VITE_DEFAULT_LOCALE)
backend_function_name ?= $(BACKEND_FUNCTION_NAME)
frontend_service_name ?= $(FRONTEND_SERVICE_NAME)
frontend_image_repository ?= $(FRONTEND_IMAGE_REPOSITORY)
cloudflare_project_name ?= $(FRONTEND_CLOUDFLARE_PROJECT_NAME)
firebase_project ?= $(FIREBASE_PROJECT_ID)
deploy_vite_api_base ?= $(DEPLOY_VITE_API_BASE)

dev:
	pnpm run dev

build:
	pnpm run build

# deploy vars
_api_base = $(if $(deploy_vite_api_base),$(deploy_vite_api_base),https://$(region)-$(project).cloudfunctions.net/$(backend_function_name)/api)
_image = ${region}-docker.pkg.dev/${project}/$(frontend_image_repository)/$(frontend_service_name)
_subs = _IMAGE=$(_image),_VITE_API_BASE=$(_api_base),_VITE_DEFAULT_LOCALE=${lang}

deploy-cloudflare:
	$(MAKE) __validate-cloudflare
	$(MAKE) __confirm-cloudflare
	printf 'VITE_API_BASE=$(_api_base)\nVITE_DEFAULT_LOCALE=${lang}\n' > .env.production
	$(MAKE) __build-static
	wrangler pages deploy build-static --project-name $(cloudflare_project_name)

deploy-firebase:
	$(MAKE) __validate-firebase
	$(MAKE) __confirm-firebase
	$(MAKE) __build-static
	firebase deploy --only hosting --project $(firebase_project)

__build-static:
	cp svelte.config.js svelte.config.js.bak
	cp svelte.config.static.js svelte.config.js
	VITE_API_BASE="$(_api_base)" VITE_DEFAULT_LOCALE="${lang}" VITE_SPA_MODE=true pnpm run build || (mv svelte.config.js.bak svelte.config.js && exit 1)
	mv svelte.config.js.bak svelte.config.js
	echo "/* /index.html 200" > build-static/_redirects

deploy-cloudrun:
	$(MAKE) __validate-cloudrun
	$(MAKE) __confirm-cloudrun
	gcloud builds submit \
	  --project ${project} \
	  --region ${region} \
	  --default-buckets-behavior=regional-user-owned-bucket \
	  --config=cloudbuild.yaml \
	  --substitutions=$(_subs) \
	  ./
	gcloud run deploy $(frontend_service_name) \
	  --project ${project} \
	  --region ${region} \
	  --image $(_image) \
	  --set-env-vars="NODE_ENV=production" \
	  --allow-unauthenticated

__validate-common:
	@if [ "$(project)" = "" ]; then echo "GCP_PROJECT_ID is not set"; exit 1; fi
	@if [ "$(region)" = "" ]; then echo "GCP_REGION is not set"; exit 1; fi
	@if [ "$(lang)" = "" ]; then echo "VITE_DEFAULT_LOCALE is not set"; exit 1; fi
	@if [ "$(backend_function_name)" = "" ]; then echo "BACKEND_FUNCTION_NAME is not set"; exit 1; fi

__validate-cloudflare: __validate-common
	@if [ "$(cloudflare_project_name)" = "" ]; then echo "FRONTEND_CLOUDFLARE_PROJECT_NAME is not set"; exit 1; fi

__validate-firebase:
	@if [ "$(firebase_project)" = "" ]; then echo "FIREBASE_PROJECT_ID is not set"; exit 1; fi

__validate-cloudrun: __validate-common
	@if [ "$(frontend_service_name)" = "" ]; then echo "FRONTEND_SERVICE_NAME is not set"; exit 1; fi
	@if [ "$(frontend_image_repository)" = "" ]; then echo "FRONTEND_IMAGE_REPOSITORY is not set"; exit 1; fi

__confirm-common:
	@printf "Continue? [y/N] "; \
	read ans; \
	case "$$ans" in y|Y|yes|YES) ;; *) echo "Deploy cancelled."; exit 1;; esac

__confirm-cloudflare:
	@echo "Deploy frontend to Cloudflare Pages with these variables:"
	@echo "  ENV_FILE=$(ENV_FILE)"
	@echo "  GCP_PROJECT_ID=$(project)"
	@echo "  GCP_REGION=$(region)"
	@echo "  BACKEND_FUNCTION_NAME=$(backend_function_name)"
	@echo "  VITE_DEFAULT_LOCALE=$(lang)"
	@echo "  VITE_API_BASE=$(_api_base)"
	@echo "  FRONTEND_CLOUDFLARE_PROJECT_NAME=$(cloudflare_project_name)"
	@$(MAKE) __confirm-common

__confirm-firebase:
	@echo "Deploy frontend to Firebase Hosting with these variables:"
	@echo "  ENV_FILE=$(ENV_FILE)"
	@echo "  FIREBASE_PROJECT_ID=$(firebase_project)"
	@$(MAKE) __confirm-common

__confirm-cloudrun:
	@echo "Deploy frontend to Cloud Run with these variables:"
	@echo "  ENV_FILE=$(ENV_FILE)"
	@echo "  GCP_PROJECT_ID=$(project)"
	@echo "  GCP_REGION=$(region)"
	@echo "  BACKEND_FUNCTION_NAME=$(backend_function_name)"
	@echo "  VITE_DEFAULT_LOCALE=$(lang)"
	@echo "  VITE_API_BASE=$(_api_base)"
	@echo "  FRONTEND_SERVICE_NAME=$(frontend_service_name)"
	@echo "  FRONTEND_IMAGE_REPOSITORY=$(frontend_image_repository)"
	@echo "  IMAGE=$(_image)"
	@$(MAKE) __confirm-common
