import pytest

from harness import variants, Runner
from go import AmbassadorTest

t = Runner("ambassador-tests", variants(AmbassadorTest))
