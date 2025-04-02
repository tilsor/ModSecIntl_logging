/*
Package logging handles the logging of information to the WACE log
file.
*/
package logging

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// TODOs:
//  - Add support for logging to RSYSLOG configurable by the wace config

// LogLevel indicates the criticality of a message, either error,
// warning or debug.
type LogLevel int

const (
	// ERROR logs errors and other critical information.
	ERROR LogLevel = iota
	// WARN logs unexpected or unusual situations that should be
	// recoverable.
	WARN
	// INFO logs interesting expected events (eg: successfully loaded a model plugin)
	INFO
	// DEBUG logs everything for debugging.
	DEBUG
)

func (ll LogLevel) String() string {
	switch ll {
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// StringToLogLevel converts a string to the corresponding LogLevel value
func StringToLogLevel(textLevel string) (LogLevel, error) {
	switch textLevel {
	case "ERROR":
		return ERROR, nil
	case "WARN":
		return WARN, nil
	case "INFO":
		return INFO, nil
	case "DEBUG":
		return DEBUG, nil
	}
	return -1, errors.New("invalid log level " + textLevel)
}

// The Logging struct holds the configured logged information.
type Logging struct {
	level LogLevel

	transactionLevel   LogLevel
	transactionBuffers map[string]*bytes.Buffer
	transactionMutex   sync.RWMutex
}

var logInstance *Logging

// Get returns or creates the unique log instance of logging
func Get() *Logging {
	if logInstance == nil {
		logInstance = new(Logging)
		logInstance.level = INFO
		logInstance.transactionLevel = WARN
		logInstance.transactionBuffers = make(map[string]*bytes.Buffer)
	}
	return logInstance
}

// LoadLoggerWriter sets up everything for the logging inside the given buffer
func (l *Logging) LoadLoggerWriter(logBuffer io.Writer, logLevel LogLevel) error {
	l.level = logLevel
	log.SetOutput(logBuffer)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println()
	log.Println("-----WACE started-----")
	return nil
}

// LoadLogger loads the logging file and sets up everything for the
// logging inside the log file
func (l *Logging) LoadLogger(logPath string, logLevel LogLevel) error {
	fh, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return l.LoadLoggerWriter(fh, logLevel)
}

// Println writes a message to the log if the given level is lower
// than the configured max level.
func (l *Logging) Println(level LogLevel, msg string) {
	if level <= l.level {
		log.Println(msg)
	}
}

// Printf writes a message to the log if the given level is lower than
// the configured max level. Arguments are handled as in fmt.Printf.
func (l *Logging) Printf(level LogLevel, format string, v ...interface{}) {
	if level <= l.level {
		log.Printf(format, v...)
	}
}

// StartTransaction creates a new buffer to log transaction
// information to eventually send to the WAF.
func (l *Logging) StartTransaction(transactionID string) {
	l.transactionMutex.Lock()
	if _, exists := l.transactionBuffers[transactionID]; !exists {
		l.transactionBuffers[transactionID] = bytes.NewBufferString("")
	}
	l.transactionMutex.Unlock()
}

// TPrintln writes a level message to the log and transaction buffer.
// It only writes to the log if the level is lower than the configured
// max level. It only writes ERROR and WARN messages to the
// transaction buffer.
func (l *Logging) TPrintln(level LogLevel, transactionID, msg string) {
	l.Println(level, "| "+transactionID+" | "+msg)

	if level <= l.transactionLevel {
		l.transactionMutex.RLock()
		buff, exists := l.transactionBuffers[transactionID]
		l.transactionMutex.RUnlock()
		if exists {
			buff.WriteString(msg)
		} else {
			l.Printf(WARN, "Cannot find transaction %s logging buffer", transactionID)
		}
	}
}

// TPrintf writes a level message to the log and transaction buffer.
// It only writes to the log if the level is lower than the configured
// max level. It only writes ERROR and WARN messages to the
// transaction buffer. Arguments are handled as in fmt.Printf.
func (l *Logging) TPrintf(level LogLevel, transactionID, format string, v ...interface{}) {
	l.Printf(level, "| "+transactionID+" | "+format, v...)

	if level <= l.transactionLevel {
		l.transactionMutex.RLock()
		buff, exists := l.transactionBuffers[transactionID]
		l.transactionMutex.RUnlock()
		if exists {
			buff.WriteString(fmt.Sprintf(format, v...))
		} else {
			l.Printf(WARN, "Cannot find transaction %s logging buffer", transactionID)
		}
	}
}

// EndTransaction returns the logging buffer for the transaction
func (l *Logging) EndTransaction(transactionID string) []byte {
	l.transactionMutex.Lock()
	res := l.transactionBuffers[transactionID].Bytes()
	delete(l.transactionBuffers, transactionID)
	l.transactionMutex.Unlock()
	return res
}
