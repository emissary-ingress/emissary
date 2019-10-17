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

release-prep:
	bash releng/release-prep.sh

release-preflight:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		printf "'make release' can only be run for commit tagged with 'vX.Y.Z'!\n"; \
		exit 1; \
	fi
ambassador-release.docker: release-preflight $(WRITE_IFCHANGED)
	docker pull $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)-rc-latest
	docker image inspect $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)-rc-latest --format='{{.Id}}' | $(WRITE_IFCHANGED) $@
release: ambassador-release.docker.push.release
release: SCOUT_APP_KEY=app.json
release: STABLE_TXT_KEY=stable.txt
release: update-aws

release-preflight-rc:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]+$$ ]]; then \
		printf "'make release-rc' can only be run for commit tagged with 'vX.Y.Z-rcN'!\n"; \
		exit 1; \
	fi
release-rc: release-preflight-rc
release-rc: ambassador.docker.push.release-rc
release-rc: SCOUT_APP_KEY = testapp.json
release-rc: STABLE_TXT_KEY = teststable.txt
release-rc: update-aws

release-preflight-ea:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+-ea[0-9]+$$ ]]; then \
		printf "'make release-ea' can only be run for commit tagged with 'vX.Y.Z-eaN'!\n"; \
		exit 1; \
	fi
release-rc: release-preflight-ea
release-ea: ambassador.docker.push.release-ea
release-ea: SCOUT_APP_KEY = earlyapp.json
release-ea: STABLE_TXT_KEY = earlystable.txt
release-ea: update-aws
