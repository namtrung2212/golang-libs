package Logger

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var globalLog *StandardAsyncLogger
var lock sync.RWMutex

// InitGlobalLogger initialize global logger
func InitGlobalLogger(logDir, fileName string, maxSize, logLevel int) error {
	lock.Lock()
	defer lock.Unlock()

	var err error
	if globalLog, err = newStandardAsyncLogger(logDir, fileName, maxSize, logLevel); err != nil {
		return err
	}

	return nil
}

// GetGlobalLogger get global logger
func GetGlobalLogger() *StandardAsyncLogger {
	lock.RLock()
	defer lock.RUnlock()

	return globalLog
}

// newStandardAsyncLogger create new standard async logger
func newStandardAsyncLogger(logDir, fileName string, maxSize, logLevel int) (*StandardAsyncLogger, error) {
	// Check file path
	filePath := filepath.Clean(filepath.Join(logDir, fileName))
	if !strings.HasPrefix(filePath, logDir) {
		return nil, fmt.Errorf("Incorrect log file path: %v", filePath)
	}

	// Init logger
	logger := &StandardAsyncLogger{
		Logger: &AsyncFileLogger{
			Filename:   filePath,
			MaxSize:    maxSize, // megabytes
			Separator:  ";;",
			MaxBackups: 61,
			MaxAge:     61, //days
		},
		LogLevel: logLevel,
		oldDay:   time.Now().Day(),
	}
	logger.Logger.ActiveAsyncWriter()

	return logger, nil
}

// StandardAsyncLogger standard logger, a wrapper over async file logger
type StandardAsyncLogger struct {
	Logger   *AsyncFileLogger
	LogLevel int

	lock   sync.RWMutex
	oldDay int
}

func (l *StandardAsyncLogger) isDayChange() bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.oldDay != time.Now().Day()
}

// AutoRotateByDay do auto rotate by day
func (l *StandardAsyncLogger) AutoRotateByDay() error {
	if l.isDayChange() {
		l.lock.Lock()
		defer l.lock.Unlock()

		l.oldDay = time.Now().Day()
		return l.Logger.Rotate()
	}

	return nil
}

// InfoLog do information log
func (l *StandardAsyncLogger) InfoLog(message interface{}) {
	l.AutoRotateByDay()
	if l.LogLevel&1 == 1 {
		l.Logger.AsyncWriteWithTime(fmt.Sprintf("[Info];;%+v", message))
	}
}

// ErrorLog do error log
func (l *StandardAsyncLogger) ErrorLog(message interface{}) {
	l.AutoRotateByDay()
	if (l.LogLevel>>1)&1 == 1 {
		l.Logger.AsyncWriteWithTime(fmt.Sprintf("[Error];;%+v", message))
	}
}

// SysInfoLog do system information log
func (l *StandardAsyncLogger) SysInfoLog(message interface{}) {
	l.AutoRotateByDay()
	l.Logger.AsyncWriteWithTime(fmt.Sprintf("[SysInfo];;%+v", message))
}

// SysErrorLog do system error log
func (l *StandardAsyncLogger) SysErrorLog(message interface{}) {
	l.AutoRotateByDay()
	l.Logger.AsyncWriteWithTime(fmt.Sprintf("[SysError];;%+v", message))
}

func WriteLog(message interface{}) {

	fmt.Println("[SysError];;%+v", message)

}
