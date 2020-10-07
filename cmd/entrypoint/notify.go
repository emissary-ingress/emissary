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
)

func notifyReconfigWebhooks(ctx context.Context) {
	// XXX: last N snapshots?
	snapshotUrl := url.QueryEscape("http://localhost:9696/snapshot")

	needDiagdNotify := true
	needSidecarNotify := true

	for {
		if notifyWebhookUrl(ctx, "diagd", fmt.Sprintf("%s?url=%s", GetEventUrl(), snapshotUrl)) {
			needDiagdNotify = false
		}

		if IsEdgeStack() {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, xurl, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			// We couldn't succesfully connect to the sidecar, probably because it hasn't
			// started up yet, so we log the error and return false to signal retry.
			log.Println(err)
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
