#!python

import sys

import yaml

hosts = list(yaml.safe_load_all(open(sys.argv[1], "r")))

authority_permutations = [
    ( "a-none", None ),
    ( "a-exact-no-match", { ":authority": "no-match" } ),
    ( "a-exact-match",    { ":authority": ""})
]

baseYAML = '''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: %(name)s
spec:
  prefix: %(prefix)s
  service: %(service)s
'''

def makeMapping(name):
    return yaml.safe_load(baseYAML % dict(name=name, prefix=f"/{name}/", service=name))
 
mappings = [ makeMapping("base-mapping") ]

m = makeMapping("exact-authority-no-match")
m["spec"]["headers"] = { ":authority": "no-match.example.com" }
mappings.append(m)

m = makeMapping("regex-authority-no-match")
m["spec"]["headers"] = { ":authority": "no-match.*\\.example\\.com", "regex": True }
mappings.append(m)

for host in hosts:
    if host["kind"] != "Host":
        continue
        
    name = host["metadata"]["name"]
    hostname = host["spec"]["hostname"]

    m = makeMapping(f"exact-authority-{name}")
    m["spec"]["headers"] = { ":authority": hostname }
    mappings.append(m)

    m = makeMapping(f"regex-authority-{name}")
    h, d = hostname.split(".", 1)
    d = d.replace(".", "\\.")

    m["spec"]["headers"] = { ":authority": f"{h}.*\\.{d}", "regex": True }
    mappings.append(m)

print(yaml.safe_dump_all(mappings))


