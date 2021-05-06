check:
	isort rel-* .
	@PS4=; set -ex; for file in rel-*; do mypy --strict "$$file" .; done
	yapf --recursive --in-place rel-* .
	pytest
.PHONY: check
