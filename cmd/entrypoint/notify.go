package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
func notifyWebhookUrl(ctx context.Context, name, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// We couldn't succesfully connect to the sidecar, probably because it hasn't
		// started up yet, so we log the error and return false to retry.
		log.Println(err)
		return false
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
