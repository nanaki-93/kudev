package logging

import (
	"flag"
	"sync"

	"k8s.io/klog/v2"
)

type LoggerInterface interface {
	Info(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	WithValues(keysAndValues ...interface{}) LoggerInterface
}
type Logger struct {
	klog.Logger
}

var _ LoggerInterface = (*Logger)(nil)

var (
	globalLogger LoggerInterface
	once         sync.Once
	mutex        sync.RWMutex
)

func InitLogger(debug bool) LoggerInterface {
	once.Do(func() {
		globalLogger = Init(debug)
	})
	return globalLogger
}

// Get returns the global logger instance, initializing if needed
func Get() LoggerInterface {
	once.Do(func() {
		globalLogger = Init(false)
	})
	mutex.RLock()
	defer mutex.RUnlock()
	return globalLogger
}

// SetLogger sets the global logger (for testing with mocks)
func SetLogger(l LoggerInterface) {
	mutex.Lock()
	defer mutex.Unlock()
	globalLogger = l
}

// ResetLogger resets the singleton for testing
func ResetLogger() {
	mutex.Lock()
	defer mutex.Unlock()
	globalLogger = nil
	once = sync.Once{}
}

// Config holds logging configuration
type Config struct {
	Level      int // 0=errors, 1=info, 4=debug
	Pretty     bool
	Structured bool
}

// DefaultConfig returns default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      0,
		Pretty:     true,
		Structured: false,
	}
}

func Init(debug bool) *Logger {
	klog.InitFlags(nil)
	klog.SetOutput(nil)
	klog.SetLogger(klog.NewKlogr())

	verbosity := "0"
	if debug {
		verbosity = "4"
	}
	if err := flag.Set("v", verbosity); err != nil {
		panic("Error during setting the log verbosity:" + err.Error())
	}

	flag.Parse()
	return &Logger{
		Logger: klog.Background(),
	}
}

func (l *Logger) Get() klog.Logger {
	return l.Logger
}
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}

func (l *Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.Logger.Error(err, msg, keysAndValues...)
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.V(4).Info(msg, keysAndValues...)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.Info("[WARN] "+msg, keysAndValues...)
}

func (l *Logger) WithValues(keysAndValues ...interface{}) LoggerInterface {
	return &Logger{
		Logger: l.Logger.WithValues(keysAndValues...),
	}
}
