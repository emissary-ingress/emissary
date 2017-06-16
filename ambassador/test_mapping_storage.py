import json

from s import AmbassadorStore
s = AmbassadorStore()

def kill_mappings(s):
    rc = s.fetch_all_mappings()
    if rc:
        for mapping in rc.mappings:
            rc2 = s.delete_mapping(mapping['name'])
            if not rc2:
                print("deleting %s: %s" % (mapping, rc2))
                break

def kill_modules(s):
    rc = s.fetch_all_modules()
    if rc:
        for module_name in rc.modules.keys():
            rc2 = s.delete_module(module_name)
            if not rc2:
                print("deleting %s: %s" % (module_name, rc2))
                break

def show_mappings(s):
    rc = s.fetch_all_mappings()
    if rc:
        for mapping in rc.mappings:
            print(mapping)

def show_modules(s):
    rc = s.fetch_all_modules()
    if rc:
        for module in rc.modules:
            print(module)

def checkmapping(rc, what, name, prefix, service, rewrite, modules):
    print("FETCH %s: %s" % (what, rc))
    assert rc, what
    assert rc.name == name, "%s: check name" % what
    assert rc.prefix == prefix, "%s: check prefix" % what
    assert rc.service == service, "%s: check service" % what
    assert rc.rewrite == rewrite, "%s: check rewrite" % what
    assert rc.modules == modules, "%s: check modules" % what

def checkmodule(rc, what, module_name, module_data):
    print("FETCH %s: %s" % (what, rc))
    assert rc, what
    assert rc.module_name == module_name, "%s: check module_name" % what
    assert rc.module_data == module_data, "%s: check module_data" % what

kill_mappings(s)
kill_modules(s)

assert s.store_mapping('xxxx', '/alice/', 'alicesvc', '/ally/', {}), "store /alice/"

show_mappings(s)
show_modules(s)

checkmapping(s.fetch_mapping('xxxx'), "fetch /alice/",
          'xxxx', '/alice/', 'alicesvc', '/ally/', {})

rc = s.delete_mapping('xxxx')
assert rc.deleted == 1
assert rc.modules_deleted == 0
assert rc, "delete /alice/"

rc = s.delete_mapping('xxxx')
assert rc.deleted == 0
assert rc.modules_deleted == 0
assert rc, "delete /alice/ again"

assert not s.fetch_mapping('xxxx'), "fetch /alice/ after delete"

basicAuth = { 'auth-service': 'ambassador' }
tlsAuth = True
mods = { 'BasicAuth': basicAuth, 'TLSAuth': tlsAuth }

kill_mappings(s)
kill_modules(s)

rc = s.store_mapping('xxxx', '/alice/', 'alicesvc', '/ally/', mods)
print("STORE MODS: %s" % rc)
assert rc, "store Alice with mods"

show_mappings(s)
show_modules(s)

checkmapping(s.fetch_mapping('xxxx'), "fetch Alice with mods",
             'xxxx', '/alice/', 'alicesvc', '/ally/', mods)

rc = s.delete_mapping_module('xxxx', 'TLSAuth')
print(rc)
assert rc, "delete TLSAuth"
assert rc.modules_deleted == 1

rc = s.delete_mapping_module('xxxx', 'TLSAuth')
print(rc)
assert rc, "delete TLSAuth again"
assert rc.modules_deleted == 0

show_mappings(s)
show_modules(s)

checkmapping(s.fetch_mapping('xxxx'), "fetch Alice with one mod",
             'xxxx', '/alice/', 'alicesvc', '/ally/', { 'BasicAuth': basicAuth })

assert s.store_mapping_module('xxxx', 'TLSAuth', tlsAuth), "restore TLSAuth"

checkmapping(s.fetch_mapping('xxxx'), "fetch Alice with restored TLSAuth",
             'xxxx', '/alice/', 'alicesvc', '/ally/', mods)

assert s.store_mapping_module('xxxx', 'BasicAuth', {'password': 'hacked!'}), "overwrite BasicAuth"

mods2 = dict(mods)
mods2['BasicAuth'] = { 'password': 'hacked!' }

checkmapping(s.fetch_mapping('xxxx'), "fetch Alice with new password",
             'xxxx', '/alice/', 'alicesvc', '/ally/', mods2)
