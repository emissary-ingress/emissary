package entrypoint

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

func snapshotServer(ctx context.Context, snapshot *atomic.Value) {
	http.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		w.Write(snapshot.Load().([]byte))
	})
	s := &http.Server{Addr: "localhost:9696"}
	go func() {
		log.Println(s.ListenAndServe())
	}()
	<-ctx.Done()
	tctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := s.Shutdown(tctx)
	if err != nil {
		panic(err)
	}
}
