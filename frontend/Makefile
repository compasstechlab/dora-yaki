dev:
	pnpm run dev

build:
	pnpm run build

# deploy vars
_api_base = https://$(region)-$(project).cloudfunctions.net/dora-yaki-backend/api
_image = ${region}-docker.pkg.dev/${project}/cloud-run-source-deploy/dora-yaki-frontend
_subs = _IMAGE=$(_image),_VITE_API_BASE=$(_api_base),_VITE_DEFAULT_LOCALE=${lang}

deploy-cloudflare:
	@if [ "$(project)" = "" ]; then echo "project is not set"; exit 1; fi
	@if [ "$(region)" = "" ]; then echo "region is not set"; exit 1; fi
	printf 'VITE_API_BASE=$(_api_base)\nVITE_DEFAULT_LOCALE=${lang}\n' > .env.production
	$(MAKE) __build-static
	wrangler pages deploy build-static --project-name dora-yaki-frontend

deploy-firebase:
	@if [ "$(project)" = "" ]; then echo "project is not set"; exit 1; fi
	$(MAKE) __build-static
	firebase deploy --only hosting --project ${project}

__build-static:
	cp svelte.config.js svelte.config.js.bak
	cp svelte.config.static.js svelte.config.js
	VITE_SPA_MODE=true pnpm run build || (mv svelte.config.js.bak svelte.config.js && exit 1)
	mv svelte.config.js.bak svelte.config.js
	echo "/* /index.html 200" > build-static/_redirects

deploy-cloudrun:
	@if [ "$(project)" = "" ]; then echo "project is not set"; exit 1; fi
	@if [ "$(region)" = "" ]; then echo "region is not set"; exit 1; fi
	gcloud builds submit \
	  --project ${project} \
	  --region ${region} \
	  --default-buckets-behavior=regional-user-owned-bucket \
	  --config=cloudbuild.yaml \
	  --substitutions=$(_subs) \
	  ./
	gcloud run deploy dora-yaki-frontend \
	  --project ${project} \
	  --region ${region} \
	  --image $(_image) \
	  --set-env-vars="NODE_ENV=production" \
	  --allow-unauthenticated
