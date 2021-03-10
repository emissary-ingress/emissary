THIS IS OUT OF DATE
===================

This really hasn't been updated for awhile. Sorry about that. 

To use `copy-gold`:

1. Do a full KAT run:
   1. Get a cluster.
   2. Set up `DEV_KUBECONFIG` and `DEV_REGISTRY` to access your cluster.
   3. `make push pytest KAT_RUN_MODE=envoy`
   4. **MAKE SURE YOU GET A CLEAN TEST RUN.** This is super-critical.

2. Run `copy-gold /tmp/gold` to copy the snapshot directories of all the Ambassadors running in KAT back to your system in the `/tmp/gold` directory.

3. Compare `/tmp/gold` to `python/tests/gold` and see what, if any, changes have happened.
   1. New directories should appear **only** if you've added tests.
   2. Old directories should disappear **only** if you've removed tests.
   3. Changes to `aconf.json` or `ir.json` files should happen **only** if you've changed test inputs.
   4. Changes to `envoy.json` files should **really** happen **only** if you've changed things.

4. If there are **any** unexplained changes, **STOP**. Figure out what's going on, fix it, and rerun the steps above.

5. Once there are **no** unintended changes, copy everything from `/tmp/gold` to `python/tests/gold`.

Original Notes
--------------

*Note*: `flynn/dev/watt` has the watt binary integrated into `entrypoint.sh` and such, so if
you build Docker images from there you'll get watt running too. The 'copy snapshots and use them
to mock stuff' bit is still very very relevant.

*Note*: There's also the `test-dump.py` program that basically acts like `diagd`, but just 
works on a snapshot file directly and spits out AConf, IR, and V2 JSON files.


Some notes on the dev loop I've been using working on watt integration:

1. Fire up a Kubernetes cluster (I used minikube for this, actually). Load it up with an Ambassador
   (in this case, one that I built, but basically 0.52.1). `kubectl cp` the binary for the new discovery 
   thing - `watt` - in place.

   `watt` basically works like the existing `kubewatch`: it constructs snapshots of relevant resource
   sets and gives Ambassador a way to grab the snapshots. This means that I can

2. Register some services with Consul, grab a snapshot from `watt`, and save it to disk. Repeat a few
   times, tweaking various things, so that I have a few sets of snapshots.

3. `kubectl cp` my snapshots back to my laptop (a Mac).

Now for the fun(?) bit: what do these things really look like? How do the data structures act as I 
tweak things? How can I manipulate them into the internal Ambassador-native resources that I actually
need? I could do this by constantly rebuilding Ambassador, pushing Docker images, and updating my
Kube cluster, but that’s _way_ too slow for the inner loop. Instead:

4. Fire up a faked version of `watt` that’s literally a Flask app serving one of my snapshot files, 
   then run `diagd` by hand.

5. `diagd` is the (increasingly-misnamed) piece of Ambassador that basically embodies the compiler
   part of Ambassador: it takes these snapshots and generates the Envoy config. Running it by hand
   locally means that I can feed it snapshots from my fake `watt` and watch what it does, then make
   a change and try again.

My inner development loop is now “make a change in some Python code, hit up-arrow RETURN in two
shell windows, see if things make sense.” This is pretty quick. Once I think it’s working correctly,
yeah, _then_ I have to build images and `kubectl apply`, but my first test cluster is `minikube`
with three pods in it, so that’s pretty quick.

So that’s how we do things at Datawire: figure out how to isolate pieces so that we _don’t_ have to
suffer with bringing clusters up and down, etc., until we’re ready to let CI have its evil way with
a new feature. It’s not nearly as smooth as we’d like for Ambassador, but it’s getting better. :)
