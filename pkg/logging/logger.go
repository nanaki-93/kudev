package logging

import (
	"flag"

	"k8s.io/klog/v2"
)

func Init(debug bool) {
	klog.InitFlags(nil)
	klog.SetOutput(nil)
	klog.SetLogger(klog.NewKlogr())
	if debug {
		if err := flag.Set("v", "4"); err != nil {
			panic("Error during setting the log verbosity:" + err.Error())
		}
	} else {
		if err := flag.Set("v", "0"); err != nil {
			panic("Error during setting the log verbosity:" + err.Error())
		}
	}

	flag.Parse()

}

func Get() klog.Logger {
	return klog.Background()
}
func Info(msg string, keysAndValues ...interface{}) {
	klog.Background().Info(msg, keysAndValues...)
}

func Error(err error, msg string, keysAndValues ...interface{}) {
	klog.Background().Error(err, msg, keysAndValues...)
}

func Debug(msg string, keysAndValues ...interface{}) {
	klog.Background().V(4).Info(msg, keysAndValues...)
}

func Warn(msg string, keysAndValues ...interface{}) {
	klog.Background().Info("[WARN] "+msg, keysAndValues...)
}

func WithValues(keysAndValues ...interface{}) klog.Logger {
	return klog.Background().WithValues(keysAndValues...)
}

// ============================================================
// Logging Configuration
// ============================================================

// LogConfig holds logging configuration.
type LogConfig struct {
	// Level: 0=errors, 1=info, 4=debug, 6=verbose
	Level int

	// Pretty: pretty-print output (for human consumption)
	Pretty bool

	// Structured: output structured JSON (for log aggregation)
	Structured bool
}

// DefaultLogConfig returns default logging configuration.
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      0, // Errors only
		Pretty:     true,
		Structured: false,
	}
}
