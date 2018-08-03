import json, os, pytest

import harness
from harness import variants, Result
from go import AmbassadorTest
from parser import dump

def label(yaml, scope):
    for obj in yaml:
        md = obj["metadata"]
        if "labels" not in md: md["labels"] = {}
        obj["metadata"]["labels"]["scope"] = scope
    return yaml


class Root:

    def __init__(self, tests):
        self.tests = tests
        self.filtered = []
        self.done = False

    def setup(self):
        if not self.done:
            self._setup_k8s()
            self._query()
            self.done = True

    def _setup_k8s(self):
        yaml = ""
        for v in self.tests:
            yaml += dump(label(v.assemble("*"), "poc-test")) + "\n"
        if os.path.exists("/tmp/k8s.yaml"):
            with open("/tmp/k8s.yaml") as f:
                prev_yaml = f.read()
        else:
            prev_yaml = None

        if yaml != prev_yaml:
            with open("/tmp/k8s.yaml", "w") as f:
                f.write(yaml)
            # XXX: better prune selector label
            os.system("kubectl apply --prune -l scope=poc-test -f /tmp/k8s.yaml")

    def _query(self):
        queries = []
        byid = {}
        for v in self.tests:
            for t in v.traversal:
                if isinstance(t, harness.Test):
                    t.pending = []
                    t.queried = []
                    t.results = []
                    for q in t.queries():
                        q.parent = t
                        t.pending.append(q)
                        queries.append(q)
                        byid[id(q)] = q

        with open("/tmp/urls.json", "w") as f:
            json.dump([{"test": q.parent.path, "id": id(q), "url": q.url} for q in queries], f)
        os.system("go run client.go -input /tmp/urls.json -output /tmp/results.json 2> /tmp/client.log")
        with open("/tmp/results.json") as f:
            results = json.load(f)

        for r in results:
            res = r["result"]
            q = byid[r["id"]]
            result = Result(q, res)
            q.parent.queried.append(q)
            q.parent.results.append(result)
            q.parent.pending.remove(q)


root = Root(tuple(v.instantiate() for v in variants(AmbassadorTest)))
params = [t for r in root.tests for t in r.traversal if isinstance(t, harness.Test)]

@pytest.mark.parametrize("t", params, ids=[t.path for t in params])
def test(t):
    root.setup()
    # XXX: should make these individual tests somehow
    for r in t.results:
        r.check()
    t.check()
