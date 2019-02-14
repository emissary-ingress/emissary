SPHINX=pipenv run sphinx-build -W . _build *.py *.rst

html:
	$(SPHINX) -b html

doctest:
	$(SPHINX) -b doctest

clean:
	rm -rf _build
