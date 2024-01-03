package main

import (
	"fmt"
	"os"

	crdAll "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext"
	"github.com/emissary-ingress/emissary/v3/pkg/utils"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "error: expected exactly one argument, got %d\n", len(os.Args))
		fmt.Fprintf(os.Stderr, "Usage: apiext {service-name}\n")
		os.Exit(2)
	}

	logger := setupLogger()
	defer logger.Sync() //nolint:errcheck

	version := utils.GetVersion()
	serviceName := os.Args[1]
	logger.Info("starting Emissary-ingress apiext webhook conversion server",
		zap.String("version", version),
		zap.String("svcName", serviceName),
	)

	options := []apiext.WebhookOption{}

	if os.Getenv("DISABLE_CRD_MANAGEMENT") != "" {
		logger.Info("disabling webhook CRD Management, CustomResourceDefinition's will not be patched with the CA Cert")
		options = append(options, apiext.WithDisableCRDPatchManagement())
	}
	if os.Getenv("DISABLE_CA_MANAGEMENT") != "" {
		logger.Info("disabling webhook CA Management, the root CA Cert will be managed externally")
		options = append(options, apiext.WithDisableCACertManagement())
	}

	webhookServer := apiext.NewWebhookServer(logger, serviceName, options...)

	g, ctx := errgroup.WithContext(ctrl.SetupSignalHandler())
	g.Go(func() error {
		scheme := crdAll.BuildScheme()
		return webhookServer.Run(ctx, scheme)
	})

	if err := g.Wait(); err != nil {
		logger.Error("an error occurred during shutdown", zap.Error(err))
	}

	logger.Info("emissary-ingress apiext server has shutdown")
}

func setupLogger() *zap.Logger {
	var level string
	if level = os.Getenv("AES_LOG_LEVEL"); level == "" {
		level = "info"
	}

	logLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		logLevel = zap.NewAtomicLevel()
		logLevel.SetLevel(zapcore.InfoLevel)
		fmt.Println("error setting up logger, AES_LOG_LEVEL is not a valid value, defaulting to log level \"info\". supported values are: debug;info;warn;error;dpanic;panic;fatal")
	}

	cfg := zap.Config{
		Encoding:         "console",
		Level:            logLevel,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	logger, _ := cfg.Build()
	return logger
}
