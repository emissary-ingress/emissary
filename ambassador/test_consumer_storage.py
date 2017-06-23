import json

from s import AmbassadorStore
s = AmbassadorStore()

def killemall(s):
    rc = s.fetch_all_consumers()
    if rc:
        for consumer in rc.consumers:
            rc2 = s.delete_consumer(consumer['consumer_id'])
            if not rc2:
                print("deleting %s: %s" % (consumer, rc2))
                break

def showemall(s):
    rc = s.fetch_all_consumers()
    if rc:
        for consumer in rc.consumers:
            print(consumer)

def checkuser(rc, what, consumer_id, username, fullname, shortname, modules):
    print("FETCH %s: %s" % (what, rc))
    assert rc, what
    assert rc.consumer_id == consumer_id, "%s: check consumer_id" % what
    assert rc.username == username, "%s: check username" % what
    assert rc.fullname == fullname, "%s: check fullname" % what
    assert rc.shortname == shortname, "%s: check shortname" % what
    assert rc.modules == modules, "%s: check modules" % what

killemall(s)

assert s.store_consumer('xxxx', 'alice', 'Alice Rules', 'Ally', {}), "store Alice"

showemall(s)

checkuser(s.fetch_consumer(consumer_id='xxxx'), "fetch Alice",
          'xxxx', 'alice', 'Alice Rules', 'Ally', {})

rc = s.delete_consumer('xxxx')
assert rc.deleted == 1
assert rc.modules_deleted == 0
assert rc, "delete Alice"

rc = s.delete_consumer('xxxx')
assert rc.deleted == 0
assert rc.modules_deleted == 0
assert rc, "delete Alice again"

assert not s.fetch_consumer(consumer_id='xxxx'), "fetch Alice by ID after delete"
assert not s.fetch_consumer(username='alice'), "fetch Alice by username after delete"

basicAuth = { 'password': 'password' }
tlsAuth = { 'principal': '30807d50499ebf448f63e9f33a5224bc3bfc2f80f8711f38913297d4f769edbe' }
mods = { 'BasicAuth': basicAuth, 'TLSAuth': tlsAuth }

killemall(s)

rc = s.store_consumer('xxxx', 'alice', 'Alice Rules', 'Ally', mods)
print("STORE MODS: %s" % rc)
assert rc, "store Alice with mods"

showemall(s)

checkuser(s.fetch_consumer(consumer_id='xxxx'), "fetch Alice with mods",
          'xxxx', 'alice', 'Alice Rules', 'Ally', mods)

rc = s.delete_consumer_module('xxxx', 'TLSAuth')
print(rc)
assert rc, "delete TLSAuth"
assert rc.modules_deleted == 1

rc = s.delete_consumer_module('xxxx', 'TLSAuth')
print(rc)
assert rc, "delete TLSAuth again"
assert rc.modules_deleted == 0

checkuser(s.fetch_consumer(consumer_id='xxxx'), "fetch Alice with one mod",
          'xxxx', 'alice', 'Alice Rules', 'Ally', { 'BasicAuth': basicAuth })

assert s.store_consumer_module('xxxx', 'TLSAuth', tlsAuth), "restore TLSAuth"

checkuser(s.fetch_consumer(username='alice'), "fetch Alice with mods by username",
          'xxxx', 'alice', 'Alice Rules', 'Ally', mods)

assert s.store_consumer_module('xxxx', 'BasicAuth', {'password': 'hacked!'}), "overwrite BasicAuth"

mods2 = dict(mods)
mods2['BasicAuth'] = { 'password': 'hacked!' }

checkuser(s.fetch_consumer(username='alice'), "fetch Alice with new password by username",
          'xxxx', 'alice', 'Alice Rules', 'Ally', mods2)
