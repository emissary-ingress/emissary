
TAG=3

backend:
	docker build . -t rschloming/backend:${TAG}
	docker push rschloming/backend:${TAG}

SPHINX=sphinx-build -W . _build *.py *.rst

html:
	$(SPHINX) -b html

doctest:
	$(SPHINX) -b doctest

clean:
	rm -rf _build
