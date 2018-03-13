package Logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	backupTimeFormat = "2006-01-02T15-04-05.000"
	defaultMaxSize   = 100
)

// ensure we always implement io.WriteCloser
var _ io.WriteCloser = (*AsyncFileLogger)(nil)

// AsyncFileLogger based on gopkg.in/natefinch/lumberjack.v2, which is an io.WriteCloser that writes to the specified filename.
// In additional, AsyncFileLogger support async writing log, which improve performance significantly.
type AsyncFileLogger struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	Filename string `json:"filename" yaml:"filename"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `json:"maxsize" yaml:"maxsize"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `json:"maxage" yaml:"maxage"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"maxbackups" yaml:"maxbackups"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"localtime" yaml:"localtime"`

	size       int64
	file       *os.File
	mu         sync.RWMutex
	c          chan []byte
	closeState chan byte

	// Log Separator
	Separator string
}

var (
	// currentTime exists so it can be mocked out by tests.
	currentTime = time.Now

	// os_Stat exists so it can be mocked out by tests.
	os_Stat = os.Stat

	// megabyte is the conversion factor between MaxSize and bytes.  It is a
	// variable so tests can mock it out and not need to write megabytes of data
	// to disk.
	megabyte = 1024 * 1024
)

// ActiveAsyncWriter active async writer
func (l *AsyncFileLogger) ActiveAsyncWriter() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.closeAsyncWriter()

	l.c = make(chan []byte, 200)
	l.closeState = make(chan byte)
	go l.asyncWrite()
}

// closeAsyncWriter close async writer
func (l *AsyncFileLogger) closeAsyncWriter() {
	if l.c != nil {
		close(l.c)
		<-l.closeState
	}
}

// asyncWriteBytes async write which call sync write to io.Writer later
func (l *AsyncFileLogger) asyncWriteBytes(p []byte) {
	defer func() {
		if e := recover(); e != nil {
		}
	}()

	if p == nil || len(p) == 0 {
		return
	}

	l.c <- p
}

// asyncWrite do async write to file
func (l *AsyncFileLogger) asyncWrite() {
	defer func() {
		if e := recover(); e != nil {
		}
	}()

	defer func() {
		l.closeState <- 1
	}()

	for p := range l.c {
		l.Write(p)
	}
}

// Write implements io.Writer.  If a write would cause the log file to be larger
// than MaxSize, the file is closed, renamed to include a timestamp of the
// current time, and a new log file is created using the original log file name.
// If the length of the write is greater than MaxSize, an error is returned.
func (l *AsyncFileLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	writeLen := int64(len(p))
	if writeLen > l.max() {
		return 0, fmt.Errorf(
			"write length %d exceeds maximum file size %d", writeLen, l.max(),
		)
	}

	if l.file == nil {
		if err = l.openExistingOrNew(len(p)); err != nil {
			return 0, err
		}
	}

	if l.size+writeLen > l.max() {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = l.file.Write(p)
	l.size += int64(n)

	return n, err
}

// Close implements io.Closer, and closes the current logfile.
func (l *AsyncFileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.close()
}

// close closes the file if it is open.
func (l *AsyncFileLogger) close() error {
	if l.file == nil {
		return nil
	}

	// Close async writer first
	l.closeAsyncWriter()

	err := l.file.Close()
	l.file = nil
	return err
}

// Rotate causes Logger to close the existing log file and immediately create a
// new one.  This is a helper function for applications that want to initiate
// rotations outside of the normal rotation rules, such as in response to
// SIGHUP.  After rotating, this initiates a cleanup of old log files according
// to the normal rules.
func (l *AsyncFileLogger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.rotate()
}

// rotate closes the current file, moves it aside with a timestamp in the name,
// (if it exists), opens a new file with the original filename, and then runs
// cleanup.
func (l *AsyncFileLogger) rotate() error {
	if err := l.close(); err != nil {
		return err
	}

	if err := l.openNew(); err != nil {
		return err
	}

	return l.cleanup()
}

// openNew opens a new log file for writing, moving any old log file out of the
// way.  This methods assumes the file has already been closed.
func (l *AsyncFileLogger) openNew() error {
	err := os.MkdirAll(l.dir(), 0744)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}

	name := l.filename()
	mode := os.FileMode(0644)
	info, err := os_Stat(name)
	if err == nil {
		// Copy the mode off the old logfile.
		mode = info.Mode()
		// move the existing file
		newname := backupName(name, l.LocalTime)
		if err := os.Rename(name, newname); err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}

		// this is a no-op anywhere but linux
		if err := chown(name, info); err != nil {
			return err
		}
	}

	// we use truncate here because this should only get called when we've moved
	// the file ourselves. if someone else creates the file in the meantime,
	// just wipe out the contents.
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}
	l.file = f
	l.size = 0
	l.c = make(chan []byte, 200)
	l.closeState = make(chan byte)
	go l.asyncWrite()

	return nil
}

// backupName creates a new filename from the given name, inserting a timestamp
// between the filename and the extension, using the local time if requested
// (otherwise UTC).
func backupName(name string, local bool) string {
	dir := filepath.Dir(name)
	filename := filepath.Base(name)
	ext := filepath.Ext(filename)
	prefix := filename[:len(filename)-len(ext)]
	t := currentTime()
	if !local {
		t = t.UTC()
	}

	timestamp := t.Format(backupTimeFormat)
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, timestamp, ext))
}

// openExistingOrNew opens the logfile if it exists and if the current write
// would not put it over MaxSize.  If there is no such file or the write would
// put it over the MaxSize, a new file is created.
func (l *AsyncFileLogger) openExistingOrNew(writeLen int) error {
	filename := l.filename()
	info, err := os_Stat(filename)
	if os.IsNotExist(err) {
		return l.openNew()
	}
	if err != nil {
		return fmt.Errorf("error getting log file info: %s", err)
	}

	if info.Size()+int64(writeLen) >= l.max() {
		return l.rotate()
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// if we fail to open the old log file for some reason, just ignore
		// it and open a new log file.
		return l.openNew()
	}
	l.file = file
	l.size = info.Size()
	l.c = make(chan []byte, 200)
	l.closeState = make(chan byte)
	go l.asyncWrite()

	return nil
}

// genFilename generates the name of the logfile from the current time.
func (l *AsyncFileLogger) filename() string {
	if l.Filename != "" {
		return l.Filename
	}
	name := filepath.Base(os.Args[0]) + "-lumberjack.log"
	return filepath.Join(os.TempDir(), name)
}

// cleanup deletes old log files, keeping at most l.MaxBackups files, as long as
// none of them are older than MaxAge.
func (l *AsyncFileLogger) cleanup() error {
	if l.MaxBackups == 0 && l.MaxAge == 0 {
		return nil
	}

	files, err := l.oldLogFiles()
	if err != nil {
		return err
	}

	var deletes []logInfo

	if l.MaxBackups > 0 && l.MaxBackups < len(files) {
		deletes = files[l.MaxBackups:]
		files = files[:l.MaxBackups]
	}
	if l.MaxAge > 0 {
		diff := time.Duration(int64(24*time.Hour) * int64(l.MaxAge))

		cutoff := currentTime().Add(-1 * diff)

		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				deletes = append(deletes, f)
			}
		}
	}

	if len(deletes) == 0 {
		return nil
	}

	go deleteAll(l.dir(), deletes)

	return nil
}

func deleteAll(dir string, files []logInfo) {
	// remove files on a separate goroutine
	for _, f := range files {
		// what am I going to do, log this?
		_ = os.Remove(filepath.Join(dir, f.Name()))
	}
}

// oldLogFiles returns the list of backup log files stored in the same
// directory as the current log file, sorted by ModTime
func (l *AsyncFileLogger) oldLogFiles() ([]logInfo, error) {
	files, err := ioutil.ReadDir(l.dir())
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %s", err)
	}
	logFiles := []logInfo{}

	prefix, ext := l.prefixAndExt()

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := l.timeFromName(f.Name(), prefix, ext)
		if name == "" {
			continue
		}
		t, err := time.Parse(backupTimeFormat, name)
		if err == nil {
			logFiles = append(logFiles, logInfo{t, f})
		}
		// error parsing means that the suffix at the end was not generated
		// by lumberjack, and therefore it's not a backup file.
	}

	sort.Sort(byFormatTime(logFiles))

	return logFiles, nil
}

// timeFromName extracts the formatted time from the filename by stripping off
// the filename's prefix and extension. This prevents someone's filename from
// confusing time.parse.
func (l *AsyncFileLogger) timeFromName(filename, prefix, ext string) string {
	if !strings.HasPrefix(filename, prefix) {
		return ""
	}
	filename = filename[len(prefix):]

	if !strings.HasSuffix(filename, ext) {
		return ""
	}
	filename = filename[:len(filename)-len(ext)]
	return filename
}

// max returns the maximum size in bytes of log files before rolling.
func (l *AsyncFileLogger) max() int64 {
	if l.MaxSize == 0 {
		return int64(defaultMaxSize * megabyte)
	}
	return int64(l.MaxSize) * int64(megabyte)
}

// dir returns the directory for the current filename.
func (l *AsyncFileLogger) dir() string {
	return filepath.Dir(l.filename())
}

// prefixAndExt returns the filename part and extension part from the Logger's
// filename.
func (l *AsyncFileLogger) prefixAndExt() (prefix, ext string) {
	filename := filepath.Base(l.filename())
	ext = filepath.Ext(filename)
	prefix = filename[:len(filename)-len(ext)] + "-"
	return prefix, ext
}

// WriteWithTime write log with current time
func (l *AsyncFileLogger) WriteWithTime(message string) (int, error) {
	if l == nil {
		return 0, nil
	}

	return l.Write([]byte(fmt.Sprintf("%s%s%s\n", time.Now().Format(time.RFC822), l.Separator, message)))
}

// AsyncWriteWithTime write log with current time
func (l *AsyncFileLogger) AsyncWriteWithTime(message string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l == nil || l.c == nil {
		return
	}

	l.asyncWriteBytes([]byte(fmt.Sprintf("%s%s%s\n", time.Now().Format(time.RFC822), l.Separator, message)))
}

// logInfo is a convenience struct to return the filename and its embedded
// timestamp.
type logInfo struct {
	timestamp time.Time
	os.FileInfo
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].timestamp.After(b[j].timestamp)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
