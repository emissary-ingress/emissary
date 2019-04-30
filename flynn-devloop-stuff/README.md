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
