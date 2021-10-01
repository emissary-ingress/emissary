package entrypoint_test

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dlog"
	"github.com/datawire/dlib/dtime"
)

const (
	PUBLIC_KEY  string = "fake-public-key"
	PRIVATE_KEY string = "fake-private-key"
)

func fakeReadPEM(ctx context.Context, dir string, name string) ([]byte, bool) {
	if name == "key.pem" {
		return []byte(PRIVATE_KEY), true
	} else if name == "cert-chain.pem" {
		return []byte(PUBLIC_KEY), true
	} else {
		return nil, false
	}
}

type icertMetadata struct {
	t      *testing.T
	ft     *dtime.FakeTime
	icert  *entrypoint.IstioCert
	events []entrypoint.IstioCertUpdate
	mutex  sync.Mutex
}

func newICertMetadata(t *testing.T) (context.Context, *icertMetadata) {
	ctx := dlog.NewTestContext(t, false)
	ft := dtime.NewFakeTime()

	updates := make(chan entrypoint.IstioCertUpdate)

	icert := entrypoint.NewIstioCert("/tmp", "istio-test", "ambassador", updates)
	icert.SetFetchTime(ft.Now)
	icert.SetReadPEM(fakeReadPEM)

	m := &icertMetadata{t: t, ft: ft, icert: icert}
	m.events = make([]entrypoint.IstioCertUpdate, 0, 5)

	go func() {
		for {
			evt := <-updates

			m.mutex.Lock()
			m.events = append(m.events, evt)
			m.mutex.Unlock()

			if evt.Op == "update" {
				dlog.Infof(ctx, "Event handler: got update of %s", evt.Secret.ObjectMeta.Name)
			} else {
				dlog.Infof(ctx, "Event handler: got deletion")
			}
		}
	}()

	return ctx, m
}

func (m *icertMetadata) stepSec(sec int) {
	m.ft.StepSec(sec)
}

func (m *icertMetadata) check(ctx context.Context, what string, name string, deleted bool, count int) {
	m.icert.HandleEvent(ctx, name, deleted)
	time.Sleep(250 * time.Millisecond)

	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.events) != count {
		m.t.Errorf("%s: wanted event count %d, got %d", what, count, len(m.events))
	}
}

func (m *icertMetadata) checkNoSecret() {
	m.mutex.Lock()
	count := len(m.events)
	m.mutex.Unlock()

	if count > 0 {
		m.mutex.Lock()
		evt := m.events[count-1]
		m.mutex.Unlock()

		if evt.Op != "delete" {
			m.t.Errorf("wanted no live secret, got %s op?", evt.Op)
		}

		if evt.Secret != nil {
			m.t.Errorf("wanted no live secret, got %s", evt.Secret.ObjectMeta.Name)
		}
	}
}

func (m *icertMetadata) checkSecret(namespace string, publicPEM string, privatePEM string) {
	wantedSecret := &k8sresourcetypes.Secret{
		TypeMeta: kates.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "istio-test",
			Namespace: namespace,
		},
		Type: k8sresourcetypes.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": []byte(privatePEM),
			"tls.crt": []byte(publicPEM),
		},
	}

	count := len(m.events)

	if count == 0 {
		m.t.Errorf("wanted live secret, have none")
		return
	}

	m.mutex.Lock()
	evt := m.events[count-1]
	m.mutex.Unlock()

	if evt.Op != "update" {
		m.t.Errorf("wanted live secret, got %s op?", evt.Op)
	}

	if evt.Name != wantedSecret.ObjectMeta.Name {
		m.t.Errorf("wanted name %s in update, got %s", wantedSecret.ObjectMeta.Name, evt.Name)
	}

	if evt.Namespace != wantedSecret.ObjectMeta.Namespace {
		m.t.Errorf("wanted namespace %s in update, got %s", wantedSecret.ObjectMeta.Namespace, evt.Namespace)
	}

	if !reflect.DeepEqual(wantedSecret, evt.Secret) {
		// We know a priori that marshaling the secrets cannot ever fail.
		wantedSecretJSON, err1 := json.MarshalIndent(wantedSecret, "", "  ")
		secretJSON, err2 := json.MarshalIndent(evt.Secret, "", "  ")

		if (err1 != nil) || (err2 != nil) {
			// WTFO.
			m.t.Errorf("secret mismatch AND impossible errors:\n-- wanted %#v\n--- (err %s)\n-- got %#v\n--- (err %s)",
				wantedSecret, err1, evt.Secret, err2)
		} else {
			// Oh good, the thing that can't fail didn't fail.
			m.t.Errorf("secret mismatch:\n-- wanted %s\n-- got %s", wantedSecretJSON, secretJSON)
		}
	}
}

func TestIstioCertHappyBoot1(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.check(ctx, "boot foo", "/tmp/foo", false, 0)
	m.check(ctx, "boot bar", "/tmp/bar", false, 0)
	m.check(ctx, "boot root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.check(ctx, "boot cert-chain.pem", "/tmp/cert-chain.pem", false, 0)
	m.check(ctx, "boot key.pem", "/tmp/key.pem", false, 1)

	m.checkSecret("ambassador", PUBLIC_KEY, PRIVATE_KEY)
}

func TestIstioCertHappyBoot2(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.check(ctx, "boot key.pem", "/tmp/key.pem", false, 0)
	m.check(ctx, "boot foo", "/tmp/foo", false, 0)
	m.check(ctx, "boot bar", "/tmp/bar", false, 0)
	m.check(ctx, "boot root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.check(ctx, "boot cert-chain.pem", "/tmp/cert-chain.pem", false, 1)

	m.checkSecret("ambassador", PUBLIC_KEY, PRIVATE_KEY)
}

func TestIstioCertHappyNoBoot(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.stepSec(5)
	m.check(ctx, "key.pem", "/tmp/key.pem", false, 0)
	m.stepSec(1)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.stepSec(2)
	m.check(ctx, "cert-chain.pem", "/tmp/cert-chain.pem", false, 1)

	m.checkSecret("ambassador", PUBLIC_KEY, PRIVATE_KEY)
}

func TestIstioCertTooSlow1(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.stepSec(5)
	m.check(ctx, "key.pem", "/tmp/key.pem", false, 0)
	m.stepSec(5)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.stepSec(5)
	m.check(ctx, "cert-chain.pem", "/tmp/cert-chain.pem", false, 0)
}

func TestIstioCertTooSlow2(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.stepSec(5)
	m.check(ctx, "key.pem", "/tmp/key.pem", false, 0)
	m.stepSec(1)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.stepSec(5)
	m.check(ctx, "cert-chain.pem", "/tmp/cert-chain.pem", false, 0)
}

func TestIstioCertEventually(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.stepSec(5)
	m.check(ctx, "key.pem", "/tmp/key.pem", false, 0)
	m.stepSec(5)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.stepSec(1)
	m.check(ctx, "cert-chain.pem", "/tmp/cert-chain.pem", false, 0)
	m.stepSec(1)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.stepSec(2)
	m.check(ctx, "key.pem", "/tmp/key.pem", false, 1)

	m.checkSecret("ambassador", PUBLIC_KEY, PRIVATE_KEY)
}

func TestIstioCertDeletion(t *testing.T) {
	ctx, m := newICertMetadata(t)

	m.stepSec(5)
	m.check(ctx, "key.pem", "/tmp/key.pem", false, 0)
	m.checkNoSecret()

	m.stepSec(1)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", false, 0)
	m.stepSec(1)
	m.check(ctx, "cert-chain.pem", "/tmp/cert-chain.pem", false, 1)
	m.checkSecret("ambassador", PUBLIC_KEY, PRIVATE_KEY)

	m.stepSec(1)
	m.check(ctx, "root-cert.pem", "/tmp/root-cert.pem", true, 1)
	m.checkSecret("ambassador", PUBLIC_KEY, PRIVATE_KEY)

	m.stepSec(1)
	m.check(ctx, "key.pem", "/tmp/key.pem", true, 2)
	m.checkNoSecret()
}
