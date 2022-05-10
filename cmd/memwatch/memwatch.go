package main

import (
	"context"
	"os"
	"time"

	"github.com/datawire/ambassador/v2/pkg/memory"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	"github.com/sirupsen/logrus"
)

var logrusLogger *logrus.Logger

func main() {
	logrusLogger = logrus.New()
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.0000",
		FullTimestamp:   true,
	})
	logrusLogger.SetReportCaller(false)

	// filepath.Join(os.TempDir(), "memory.log")
	output, err := os.OpenFile("/proc/1/fd/1", os.O_RDWR, 0666)

	if err != nil {
		logrusLogger.Fatal(err)
		return
	}

	logrusLogger.SetOutput(output)

	logger := dlog.WrapLogrus(logrusLogger)
	ctx := dlog.WithLogger(context.Background(), logger) // early in Main()

	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
		SoftShutdownTimeout:  10 * time.Second,
		HardShutdownTimeout:  10 * time.Second,
	})

	usage := memory.GetMemoryUsage(ctx)

	group.Go("memory", func(ctx context.Context) error {
		usage.Watch(ctx)
		return nil
	})

	// cancel()
	group.Wait()
}
