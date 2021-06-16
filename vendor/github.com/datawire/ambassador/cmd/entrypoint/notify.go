package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/datawire/ambassador/v2/pkg/acp"
	"github.com/datawire/ambassador/v2/pkg/debug"

	"github.com/datawire/dlib/dlog"
)

type notable interface {
	NoteSnapshotSent()
	NoteSnapshotProcessed()
}

type noopNotable struct{}

func (_ *noopNotable) NoteSnapshotSent()      {}
func (_ *noopNotable) NoteSnapshotProcessed() {}

func notifyReconfigWebhooks(ctx context.Context, ambwatch notable) {
	notifyReconfigWebhooksFunc(ctx, ambwatch, IsEdgeStack())
}

func notifyReconfigWebhooksFunc(ctx context.Context, ambwatch notable, edgestack bool) {
	// XXX: last N snapshots?
	snapshotUrl := url.QueryEscape("http://localhost:9696/snapshot")

	needDiagdNotify := true
	needSidecarNotify := true

	// We're about to send a new snapshot to diagd. The webhook we're using for this
	// won't return, by design, until the snapshot has been processed, so first note
	// that we're sending the snapshot...
	ambwatch.NoteSnapshotSent()

	for {
		// ...then send it and wait for the webhook to return...
		if notifyWebhookUrl(ctx, "diagd", fmt.Sprintf("%s?url=%s", GetEventUrl(), snapshotUrl)) {
			needDiagdNotify = false
			// ...then note that it's been processed. This DOES NOT imply that the processing
			// was successful: it's just about whether or not diagd is making progress instead
			// of getting stuck.
			ambwatch.NoteSnapshotProcessed()
		}

		// Then go deal with the Edge Stack sidecar.
		if edgestack {
			if notifyWebhookUrl(ctx, "edgestack sidecar", fmt.Sprintf("%s?url=%s", GetSidecarUrl(), snapshotUrl)) {
				needSidecarNotify = false
			}
		} else {
			needSidecarNotify = false
		}

		select {
		case <-ctx.Done():
			return
		default:
			// XXX: find a better way to wait for diagd and/or the sidecar to spin up
			if needDiagdNotify || needSidecarNotify {
				time.Sleep(1 * time.Second)
			} else {
				return
			}
		}
	}
}

// posts to a webhook style url, logging any errors, and returning false if a retry is needed
func notifyWebhookUrl(ctx context.Context, name, xurl string) bool {
	defer debug.FromContext(ctx).Timer(fmt.Sprintf("notifyWebhook:%s", name)).Start()()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, xurl, nil)
	if err != nil {
		panic(err)
	}

	// As long as this notification is going to localhost, set the X-Ambassador-Diag-IP
	// header for it. This is only used by diagd right now, but this is the easy way
	// to deal with it.
	parsedURL, err := url.Parse(xurl)

	if err != nil {
		// This is "impossible" in that it's a blatant programming error, not
		// an error caused by the user. Panic.
		panic(fmt.Errorf("bad URL passed to notifyWebhookUrl: '%s', %v", xurl, err))
	}

	// OK, the URL parsed clean (as it *!&@*#& well should have!) so we can find
	// out if it's going to localhost. We'll do this the strict way, since these
	// URLs should be hardcoded.

	if acp.HostPortIsLocal(fmt.Sprintf("%s:%s", parsedURL.Hostname(), parsedURL.Port())) {
		// If we're speaking to localhost, we're speaking from localhost. Hit it.
		req.Header.Set("X-Ambassador-Diag-IP", "127.0.0.1")
	}

	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			// We couldn't succesfully connect to the sidecar, probably because it hasn't
			// started up yet, so we log the error and return false to signal retry.
			dlog.Errorf(ctx, err.Error())
			return false
		} else {
			// If either of the sidecars cannot successfully handle a webhook request, we
			// deliberately consider it a fatal error so that we can ensure shared fate between all
			// ambassador processes. The only known case where this occurs so far is when the diagd
			// gunicorn worker gets OOMKilled. This results in an EOF and we end up here.
			panic(err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("error reading body from %s: %v", name, err)
		} else {
			log.Printf("error notifying %s: %s, %s", name, resp.Status, string(body))
		}
	}

	// We assume the sidecars are idempotent. That means we don't want to retry even if we get
	// back a non 200 response since we would get an error the next time also and just be stuck
	// retrying forever.
	return true
}
