package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/dlib/dlog"
)

// Istio TLS certificates are annoying. They come in three parts (only two of
// which are needed), they're updated non-atomically, and we need to make sure we
// don't try to reconfigure when the parts are out of sync. Therefore, we keep
// track of the last-update time of both parts, and only update once both have
// been updated at the "same" time.

type pemReader func(ctx context.Context, dir string, name string) ([]byte, bool)
type timeFetcher func() time.Time

// IstioCert holds all the state we need to manage an Istio certificate.
type IstioCert struct {
	dir        string
	name       string // Name we'll use when generating our secret
	namespace  string // Namespace in which our secret will appear to be
	timestamps map[string]time.Time

	// How shall we read PEM files?
	readPEM pemReader

	// How shall we fetch the current time?
	fetchTime timeFetcher

	// Where shall we send updates when things happen?
	updates chan IstioCertUpdate
}

// IstioCertUpdate gets sent over the IstioCert's Updates channel
// whenever the cert changes
//
// XXX This will morph into a more general "internally-generated resource"
// thing later.
type IstioCertUpdate struct {
	Op        string        // "update" or "delete"
	Name      string        // secret name
	Namespace string        // secret namespace
	Secret    *kates.Secret // IstioCert secret
}

// NewIstioCert instantiates an IstioCert to manage a certificate that Istio
// will write into directory "dir", should have the given "name" and appear
// to live in K8s namespace "namespace", and will have updates posted to
// "updateChannel" whenever the cert changes.
//
// What's with this namespace business? Well, Ambassador may be running in
// single-namespace mode, so causing our cert to appear to be in the same
// namespace as Ambassador will just be less confusing for everyone.
//
// XXX Nomenclature is a little odd here. Istio is writing a _certificate_,
// but we're supplying it to the rest of Ambassador as a thing that looks
// like a Kubernetes TLS _Secret_ -- so we call this class an IstioCert,
// but the thing it's posting to the updateChannel includes a kates.Secret.
// Names are hard.
func NewIstioCert(dir string, name string, namespace string, updateChannel chan IstioCertUpdate) *IstioCert {
	icert := &IstioCert{
		dir:       dir,
		name:      name,
		namespace: namespace,
		fetchTime: time.Now, // default to using time.Now for time
		updates:   updateChannel,
	}

	// Default to using our default PEM reader...
	icert.readPEM = icert.defaultReadPEM

	// ...initialize the timestamp map...
	icert.timestamps = make(map[string]time.Time)

	return icert
}

// String returns a string representation of this IstioCert.
func (icert *IstioCert) String() string {
	// Our dir may be nothing, if we're just a dummy IstioCert
	// that's being used to make other logic easier. If that's the
	// case, be a little more verbose here.

	if icert.dir == "" {
		return "IstioCert (noop)"
	}

	return fmt.Sprintf("IstioCert %s", icert.dir)
}

// SetFetchTime will change the function we use to get the current time.
func (icert *IstioCert) SetFetchTime(fetchTime timeFetcher) {
	icert.fetchTime = fetchTime
}

// SetReadPEM will change the function we use to read PEM files.
func (icert *IstioCert) SetReadPEM(readPEM pemReader) {
	icert.readPEM = readPEM
}

// defaultReadPEM is the same as ioutil.ReadFile, really, but it logs for us
// if anything goes wrong.
func (icert *IstioCert) defaultReadPEM(ctx context.Context, dir string, name string) ([]byte, bool) {
	pemPath := path.Join(dir, name)

	pem, err := ioutil.ReadFile(pemPath)

	if err != nil {
		dlog.Errorf(ctx, "%s: couldn't read %s: %s", icert, pemPath, err)
		return nil, false
	}

	return pem, true
}

// getTimeFor tries to look up the timestamp we have stored for a given key,
// but it logs if it's missing (for debugging).
func (icert *IstioCert) getTimeFor(ctx context.Context, name string) (time.Time, bool) {
	then, exists := icert.timestamps[name]

	if !exists {
		dlog.Debugf(ctx, "%s: %s missing", icert, name)
		return time.Time{}, false
	}

	return then, true
}

// HandleEvent tells an IstioCert to update its internal state because a file
// in its directory has been written. If all the cert files have been updated
// closely enough in time, Update will decide that it's time to actually update
// the cert, and it'll send an IstioCertUpdate over the Updates channel.
func (icert *IstioCert) HandleEvent(ctx context.Context, name string, deleted bool) {
	// Istio writes three files into its cert directory:
	// - key.pem is the private key
	// - cert-chain.pem is the public keychain
	// - root-cert.pem is the CA root public key
	//
	// We ignore root-cert.pem, because cert-chain.pem contains it, and we
	// ignore any other name because Istio shouldn't be writing it there.
	//
	// Start by splitting the incoming name (which is really a path) into its
	// component parts, just 'cause it (mostly) makes life easier to refer
	// to things by the basename (the key) rather than the full path.
	dir := path.Dir(name)
	key := path.Base(name)

	dlog.Debugf(ctx, "%s: updating %s at %s", icert, key, name)

	if dir != icert.dir {
		// Somehow this is in the wrong directory? Toss it.
		dlog.Debugf(ctx, "%s: ignoring %s in dir %s", icert, name, dir)
		return
	}

	if (key != "key.pem") && (key != "cert-chain.pem") {
		// Someone is writing a file we don't need. Toss it.
		dlog.Debugf(ctx, "%s: ignoring %s", icert, name)
		return
	}

	// If this is a deletion...
	if deleted {
		// ...then drop the key from our timestamps map.
		delete(icert.timestamps, key)
	} else {
		// Not a deletion -- update the time for this key.
		icert.timestamps[key] = icert.fetchTime()
	}

	// Do we have both key.pem and cert-chain.pem? (It's OK to just return immediately
	// without logging, if not, because getTime logs for us.)
	kTime, kExists := icert.getTimeFor(ctx, "key.pem")
	cTime, cExists := icert.getTimeFor(ctx, "cert-chain.pem")

	bothPresent := (kExists && cExists)

	// OK. Is it time to do anything?
	if bothPresent && !deleted {
		// Maybe. Everything we need is present, but are the updates close enough
		// in time? Start by finding out which of key & cert-chain is newest, so
		// that we can find the delta between them.
		//
		// XXX Wouldn't it be nice if time.Duration had an AbsValue method?t
		var delta time.Duration

		if cTime.Before(kTime) {
			delta = kTime.Sub(cTime)
		} else {
			delta = cTime.Sub(kTime)
		}

		// OK, if the delta is more than five seconds (which is crazy long), we're done.
		//
		// Why five? Well, mostly just 'cause it's really easy to imagine these getting
		// written on different sides of a second boundary, really hard to imagine it
		// taking longer than five seconds, and really hard to imagine trying to rotate
		// certs every few seconds instead of every few minutes...

		if delta > 5*time.Second {
			dlog.Debugf(ctx, "%s: cert-chain/key delta %v, out of sync", icert, delta)
			return
		}

		// OK, times look good -- grab the JSON for this thing.
		secret, ok := icert.Secret(ctx)

		if !ok {
			// WTF.
			dlog.Debugf(ctx, "%s: cannot construct secret", icert)
		}

		// FINALLY!
		dlog.Debugf(ctx, "%s: noting update!", icert)

		go func() {
			icert.updates <- IstioCertUpdate{
				Op:        "update",
				Name:      secret.ObjectMeta.Name,
				Namespace: secret.ObjectMeta.Namespace,
				Secret:    secret,
			}

			dlog.Debugf(ctx, "%s: noted update!", icert)
		}()
	} else if deleted && !bothPresent {
		// OK, this is a deletion, and we no longer have both files. Time to
		// post the deletion.
		//
		// XXX We can actually generate _two_ deletions if both files are
		// deleted. We're not worrying about that for now.

		dlog.Debugf(ctx, "%s: noting deletion", icert)

		go func() {
			// Kind of a hack -- since we can't generate a real Secret object
			// without having the files we need, send the name & namespace from
			// icert.
			icert.updates <- IstioCertUpdate{
				Op:        "delete",
				Name:      icert.name,
				Namespace: icert.namespace,
				Secret:    nil,
			}

			dlog.Debugf(ctx, "%s: noted deletion!", icert)
		}()
	} else {
		dlog.Debugf(ctx, "%s: nothing to note", icert)
	}
}

// Secret generates a kates.Secret for this IstioCert. Since this
// involves reading PEM, it can fail, so it logs and returns a status.
func (icert *IstioCert) Secret(ctx context.Context) (*kates.Secret, bool) {
	privatePEM, privateOK := icert.readPEM(ctx, icert.dir, "key.pem")
	publicPEM, publicOK := icert.readPEM(ctx, icert.dir, "cert-chain.pem")

	if !privateOK || !publicOK {
		dlog.Errorf(ctx, "%s: read error, bailing", icert)
		return nil, false
	}

	newSecret := &kates.Secret{
		TypeMeta: kates.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      icert.name,
			Namespace: icert.namespace,
		},
		Type: kates.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": privatePEM,
			"tls.crt": publicPEM,
		},
	}

	return newSecret, true
}
