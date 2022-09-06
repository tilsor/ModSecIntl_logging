package logging

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
)

var msg1 = "Lorem ipsum dolor sit amet"
var msg2 = "Consectetur adipiscing elit"
var msgNot = "This should not appear in the log"

func generateRandomID() string {
	letters := "1234567890ABCDEF"
	id := ""
	for i := 0; i < 16; i++ {
		id += string(letters[rand.Intn(len(letters))])
	}

	return id
}

func TestUninitialized(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	l := Get()
	l.Println(ERROR, msg1)

	if !strings.Contains(buf.String(), msg1) {
		t.Errorf("log output is \"%s\", expected \"%s\"", buf.String(), msg1)
	}
}

func TestLoadLoggerInvalid(t *testing.T) {
	l := Get()
	err := l.LoadLogger("", WARN)
	if err == nil {
		t.Errorf("LoadLogger with empty path does not return an error")
	}
}

func TestLoadLogger(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "logging_test-")
	if err != nil {
		t.Errorf("cannot create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	l := Get()
	err = l.LoadLogger(tmpFile.Name(), WARN)
	if err != nil {
		t.Errorf("LoadLogger(%s) raised error: %v", tmpFile.Name(), err)
	}

	l = Get()
	l.Printf(ERROR, "%s", msg1)
	l.Println(WARN, msg2)
	l.Println(INFO, msgNot)

	logContents, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Errorf("cannot read log file: %v", err)
	}
	if !strings.Contains(string(logContents), msg1) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msg1)
	}
	if !strings.Contains(string(logContents), msg2) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msg2)
	}

	if strings.Contains(string(logContents), msgNot) {
		t.Errorf("log output contains \"%s\", but shouldn't", msgNot)
	}
}

func TestLogLevel(t *testing.T) {
	cases := []string{"ERROR", "WARN", "INFO", "DEBUG"}
	for _, c := range cases {
		if ll, err := StringToLogLevel(c); err != nil || c != ll.String() {
			t.Errorf("Error processing %s LogLevel", c)
		}
	}

	if _, err := StringToLogLevel("INVALID!"); err == nil {
		t.Errorf("Invalid LogLevel did not rise an error")
	}

}

func TestTransactionLogger(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "logging_test-")
	if err != nil {
		t.Errorf("cannot create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	l := Get()
	err = l.LoadLogger(tmpFile.Name(), DEBUG)
	if err != nil {
		t.Errorf("LoadLogger(%s) raised error: %v", tmpFile.Name(), err)
	}

	l = Get()
	transactionID := generateRandomID()
	l.StartTransaction(transactionID)
	l.TPrintln(ERROR, transactionID, msg1)
	l.TPrintf(WARN, transactionID, "%s", msg2)
	l.TPrintf(INFO, transactionID, "%s", msgNot)

	// this should not crash:
	l.TPrintln(WARN, generateRandomID(), msg1)
	l.TPrintf(ERROR, generateRandomID(), "%s", msg2)

	logContents := l.EndTransaction(transactionID)

	if !strings.Contains(string(logContents), msg1) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msg1)
	}
	if !strings.Contains(string(logContents), msg2) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msg2)
	}

	if strings.Contains(string(logContents), msgNot) {
		t.Errorf("log output contains \"%s\", but shouldn't", msgNot)
	}

	// Test standard log
	logContents, err = ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Errorf("cannot read log file: %v", err)
	}
	if !strings.Contains(string(logContents), msg1) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msg1)
	}
	if !strings.Contains(string(logContents), msg2) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msg2)
	}

	if !strings.Contains(string(logContents), msgNot) {
		t.Errorf("log output is \"%s\", should include \"%s\"", logContents, msgNot)
	}

}
