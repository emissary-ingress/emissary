import json, os, pytest, sys

import harness
from harness import variants, Root
from go import AmbassadorTest

root = Root(tuple(v.instantiate() for v in variants(AmbassadorTest)))
params = [t for r in root.tests for t in r.traversal if isinstance(t, harness.Test)]

@pytest.mark.parametrize("t", params, ids=[t.path for t in params])
def test(request, t):
    selected = set(item.callspec.getparam('t') for item in request.session.items)
    root.setup(selected)
    # XXX: should aggregate the result of url checks
    for r in t.results:
        r.check()
    t.check()
