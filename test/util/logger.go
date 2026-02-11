package util

import "github.com/nanaki-93/kudev/pkg/logging"

type MockLogger struct {
	Messages []string
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.Messages = append(m.Messages, msg)
}

func (m *MockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	m.Messages = append(m.Messages, msg)
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.Messages = append(m.Messages, msg)
}
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.Messages = append(m.Messages, msg)
}
func (m *MockLogger) WithValues(keysAndValues ...interface{}) logging.LoggerInterface {
	return &MockLogger{
		Messages: m.Messages,
	}
}
