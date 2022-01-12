from typing import Optional

import json
import logging
import os
import sys

import jsonschema
import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")


def test_schemas():
    rootdir = os.path.join(os.path.dirname(__file__), "..")

    schemadir = rootdir

    while schemadir != '/':
        if os.path.isdir(os.path.join(schemadir, 'python', 'schemas')):
            break
        schemadir = os.path.abspath(os.path.join(schemadir, '..'))

    schemadir = os.path.join(schemadir, 'python', 'schemas')

    if not os.path.isdir(schemadir):
        assert False, f"could not find python/schemas directory starting at {rootdir}"
        return

    # We have a schemadir.

    logger.info(f"schemadir {schemadir}")
    errors = 0

    for dirpath, dirs, files in os.walk(schemadir):
        for filename in files:
            if not filename.endswith('.schema'):
                continue

            schema_path = os.path.join(dirpath, filename)
            schema = None

            try:
                schema = json.load(open(schema_path, "r"))
            except OSError as e:
                logger.error(f"{schema_path}: could not read - {e}")
                errors += 1
            except json.decoder.JSONDecodeError as e:
                logger.error(f"{schema_path}: corrupt schema - {e}")
                errors += 1
            except Exception as e:
                logger.error(f"{schema_path}: unknown exception while parsing - {e}")
                errors += 1

            if not schema:
                continue

            try:
                jsonschema.Draft4Validator.check_schema(schema)
            except jsonschema.exceptions.SchemaError as e:
                logger.error(f"{schema_path}: invalid schema - {e}")
                errors += 1
            except Exception as e:
                logger.error(f"{schema_path}: unknown exception while validating - {e}")
                errors += 1

    assert errors == 0, f"Errors found: {errors}"


if __name__ == '__main__':
    pytest.main(sys.argv)
