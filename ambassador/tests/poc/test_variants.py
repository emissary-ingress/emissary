import pytest

from harness import variants, Runner
from go import AmbassadorTest

runner = Runner("poc-test", variants(AmbassadorTest))

@pytest.mark.parametrize("t", runner.tests, ids=runner.ids)
def test(request, t):
    selected = set(item.callspec.getparam('t') for item in request.session.items)
    runner.setup(selected)
    # XXX: should aggregate the result of url checks
    for r in t.results:
        r.check()
    t.check()
