package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/emissary-ingress/emissary/v3/pkg/acp"
	"github.com/emissary-ingress/emissary/v3/pkg/debug"

	"github.com/datawire/dlib/dlog"
)

type notable interface {
	NoteSnapshotSent()
	NoteSnapshotProcessed()
}

type noopNotable struct{}

func (_ *noopNotable) NoteSnapshotSent()      {}
func (_ *noopNotable) NoteSnapshotProcessed() {}

func notifyReconfigWebhooks(ctx context.Context, ambwatch notable) error {
	isEdgeStack, err := IsEdgeStack()
	if err != nil {
		return err
	}
	return notifyReconfigWebhooksFunc(ctx, ambwatch, isEdgeStack)
}

func notifyReconfigWebhooksFunc(ctx context.Context, ambwatch notable, edgestack bool) error {
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
		finished, err := notifyWebhookUrl(ctx, "diagd", fmt.Sprintf("%s?url=%s", GetEventUrl(), snapshotUrl))
		if err != nil {
			return err
		}
		if finished {
			needDiagdNotify = false
			// ...then note that it's been processed. This DOES NOT imply that the processing
			// was successful: it's just about whether or not diagd is making progress instead
			// of getting stuck.
			ambwatch.NoteSnapshotProcessed()
		}

		// Then go deal with the Edge Stack sidecar.
		if edgestack {
			finished, err := notifyWebhookUrl(ctx, "edgestack sidecar", fmt.Sprintf("%s?url=%s", GetSidecarUrl(), snapshotUrl))
			if err != nil {
				return err
			}
			if finished {
				needSidecarNotify = false
			}
		} else {
			needSidecarNotify = false
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			// XXX: find a better way to wait for diagd and/or the sidecar to spin up
			if needDiagdNotify || needSidecarNotify {
				time.Sleep(1 * time.Second)
			} else {
				return nil
			}
		}
	}
}

// posts to a webhook style url, logging any errors, and returning false if a retry is needed
func notifyWebhookUrl(ctx context.Context, name, xurl string) (bool, error) {
	defer debug.FromContext(ctx).Timer(fmt.Sprintf("notifyWebhook:%s", name)).Start()()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, xurl, nil)
	if err != nil {
		return false, err
	}

	// As long as this notification is going to localhost, set the X-Ambassador-Diag-IP
	// header for it. This is only used by diagd right now, but this is the easy way
	// to deal with it.
	parsedURL, err := url.Parse(xurl)
	if err != nil {
		// This is "impossible" in that it's a blatant programming error, not
		// an error caused by the user. Panic.
		panic(fmt.Errorf("BUG: bad URL passed to notifyWebhookUrl: '%s', %v", xurl, err))
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
			dlog.Error(ctx, err.Error())
			return false, nil
		} else {
			// If either of the sidecars cannot successfully handle a webhook request, we
			// deliberately consider it a fatal error so that we can ensure shared fate between all
			// ambassador processes. The only known case where this occurs so far is when the diagd
			// gunicorn worker gets OOMKilled. This results in an EOF and we end up here.
			return false, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			dlog.Printf(ctx, "error reading body from %s: %v", name, err)
		} else {
			dlog.Printf(ctx, "error notifying %s: %s, %s", name, resp.Status, string(body))
		}
	}

	// We assume the sidecars are idempotent. That means we don't want to retry even if we get
	// back a non 200 response since we would get an error the next time also and just be stuck
	// retrying forever.
	return true, nil
}
