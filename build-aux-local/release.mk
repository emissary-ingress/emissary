SCOUT_APP_KEY=

update-aws:
ifeq ($(AWS_ACCESS_KEY_ID),)
	@echo 'AWS credentials not configured; not updating either https://s3.amazonaws.com/datawire-static-files/ambassador/$(STABLE_TXT_KEY) or the latest version in Scout'
else
	@if [ -n "$(STABLE_TXT_KEY)" ]; then \
        printf "$(RELEASE_VERSION)" > stable.txt; \
		echo "updating $(STABLE_TXT_KEY) with $$(cat stable.txt)"; \
        aws s3api put-object \
            --bucket datawire-static-files \
            --key ambassador/$(STABLE_TXT_KEY) \
            --body stable.txt; \
	fi
	@if [ -n "$(SCOUT_APP_KEY)" ]; then \
		printf '{"application":"ambassador","latest_version":"$(RELEASE_VERSION)","notices":[]}' > app.json; \
		echo "updating $(SCOUT_APP_KEY) with $$(cat app.json)"; \
        aws s3api put-object \
            --bucket scout-datawire-io \
            --key ambassador/$(SCOUT_APP_KEY) \
            --body app.json; \
	fi
endif
.PHONY: update-aws

release-prep:
	bash releng/release-prep.sh
.PHONY: release-prep

release-preflight:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		printf "'make release' can only be run for commit tagged with 'vX.Y.Z'!\n"; \
		exit 1; \
	fi
ambassador-release.docker.stamp: release-preflight
	docker pull $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)-rc-latest
	docker image inspect $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)-rc-latest --format='{{.Id}}' > $@
release: ambassador-release.docker.push.release
	$(MAKE) SCOUT_APP_KEY=app.json STABLE_TXT_KEY=stable.txt update-aws
.PHONY: release-preflight release

release-preflight-rc:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]+$$ ]]; then \
		printf "'make release-rc' can only be run for commit tagged with 'vX.Y.Z-rcN'!\n"; \
		exit 1; \
	fi
ambassador-release-rc.docker.stamp: release-preflight-rc | ambassador.docker
	cat ambassador.docker > $@
release-rc: ambassador-release-rc.docker.push.release
	$(MAKE) SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
.PHONY: release-preflight-rc release-rc

release-preflight-ea:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+-ea[0-9]+$$ ]]; then \
		printf "'make release-ea' can only be run for commit tagged with 'vX.Y.Z-eaN'!\n"; \
		exit 1; \
	fi
ambassador-release-ea.docker.stamp: release-preflight-ea | ambassador.docker
	cat ambassador.docker > $@
release-ea: ambassador-release-ea.docker.push.release
	$(MAKE) SCOUT_APP_KEY=earlyapp.json STABLE_TXT_KEY=earlystable.txt update-aws
.PHONY: release-preflight-ea release-ea
