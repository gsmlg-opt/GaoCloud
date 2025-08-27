package log

import (
	"fmt"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	logger, _ := NewLog4jLogger("test.log", Warn, 0, 0)
	logger.Info("info message")
	logger.Debug("debug message")
	logger.Warn("warn message")
	logger.Error("error message")
	<-time.After(1 * time.Second)
	logger.Close()
}

func TestBufLogger(t *testing.T) {
	logger, logCh := NewLog4jBufLogger(2, Info)
	defer logger.Close()

	logger.Info("buf log: info message")
	logger.Debug("buf log: debug message")
	logger.Warn("buf log: warn message")
	logger.Error("buf log: error message")
	<-time.After(1 * time.Second)

	for i := 0; i < 2; i++ {
		log := <-logCh
		fmt.Printf(log)
	}
}

func TestISO3339BufLogger(t *testing.T) {
	logger, logCh := NewISO3339Log4jBufLogger(2, Info)
	defer logger.Close()

	logger.Info("buf log: info message")
	logger.Debug("buf log: debug message")
	logger.Warn("buf log: warn message")
	logger.Error("buf log: error message")
	<-time.After(1 * time.Second)

	for i := 0; i < 2; i++ {
		log := <-logCh
		fmt.Printf(log)
	}
}

func TestTermLogger(t *testing.T) {
	InitLogger(Debug)
	defer CloseLogger()
	Fatalf("good boy")
}
