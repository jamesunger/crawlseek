# Variables
REGISTRY=rg.fr-par.scw.cloud/crawlseek-cr
FRONTEND_IMAGE=$(REGISTRY)/frontend:latest
SEEKER_IMAGE=$(REGISTRY)/seeker:latest

# Directories
REACT_APP_DIR=dungeon-crawl-seed-finder
SEEKER_DIR=seeker
HELM_DIR=helm
HELM_SEEKER_DIR=$(SEEKER_DIR)/helm

# Default make goal
.PHONY: all
all: upgrade-helm upgrade-seeker
#all: build-react-app build-docker publish-frontend publish-seeker upgrade-helm upgrade-seeker

# Rule to build React app
.PHONY: build-react-app
build-react-app:
	cd $(REACT_APP_DIR) && npm install && npm run build

# Rule to build the top level Dockerfile
.PHONY: build-docker
build-docker: build-react-app
	docker build -t $(FRONTEND_IMAGE) .

# Rule to publish the frontend image
.PHONY: publish-frontend
publish-frontend: build-docker
	docker push $(FRONTEND_IMAGE)

# Rule to build seeker Dockerfile
.PHONY: build-seeker
build-seeker:
	docker build -t $(SEEKER_IMAGE) $(SEEKER_DIR)

# Rule to publish the seeker image
.PHONY: publish-seeker
publish-seeker: build-seeker
	docker push $(SEEKER_IMAGE)

# Rule to upgrade helm charts for main helm
.PHONY: upgrade-helm
upgrade-helm: publish-frontend
	helm upgrade --install frontend $(HELM_DIR) --namespace crawlseek --values $(HELM_DIR)/values.yaml
	kubectl -n crawlseek rollout restart deployment frontend-frontend

# Rule to upgrade helm charts for seeker
.PHONY: upgrade-seeker
upgrade-seeker: publish-seeker
	helm upgrade --install seeker $(HELM_SEEKER_DIR) --namespace crawlseek --values $(HELM_SEEKER_DIR)/values.yaml
	kubectl -n crawlseek rollout restart deployment seeker-seeker
