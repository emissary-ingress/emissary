package runner

import (
	"context"
	"path/filepath"
	"sync"

	// 3rd-party libraries
	"github.com/fsnotify/fsnotify"
)

func triggerOnChange(ctx context.Context, watchFile string, trigger func()) {
	file := filepath.Clean(watchFile)
	dir, _ := filepath.Split(file)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrusLogger.Errorf("Failed to create watch on %s: Changes might require a restart: %v", file, err)
	}
	defer watcher.Close()

	eventsWG := sync.WaitGroup{}
	eventsWG.Add(1)
	go func() {
		defer eventsWG.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					if trigger != nil {
						trigger()
					}
				} else if filepath.Clean(event.Name) == file &&
					event.Op&fsnotify.Remove&fsnotify.Remove != 0 {
					return
				}
			case err, ok := <-watcher.Errors:
				if ok {
					logrusLogger.Errorln(err)
				}
				return
			}
		}
	}()

	logrusLogger.Infof("Creating watch on %s", dir)
	err = watcher.Add(dir)
	if err != nil {
		logrusLogger.Errorf("Failed to create watch on %s: Changes might require a restart: %v", dir, err)
	}
	eventsWG.Wait()
}
