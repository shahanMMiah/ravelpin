package logging

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
)

type LogObject struct {
	File       *os.File
	LogHandler *slog.TextHandler
	Path       string
}

type SSEMap struct {
	Lock      sync.Mutex
	StatusMap map[string]string
}

func NewSSE() *SSEMap {
	return &SSEMap{Lock: sync.Mutex{}, StatusMap: make(map[string]string, 0)}
}

func (m *SSEMap) Add(job, status string) {

	m.Lock.Lock()
	defer m.Lock.Unlock()

	m.StatusMap[job] = status
}

func (m *SSEMap) Get(job string) (string, error) {

	m.Lock.Lock()
	defer m.Lock.Unlock()

	stat, err := m.StatusMap[job]

	if !err {
		return "", fmt.Errorf("Job doesnt exists")
	}

	return stat, nil

}
func (m *SSEMap) View() map[string]string {

	m.Lock.Lock()
	defer m.Lock.Unlock()

	return m.StatusMap

}

func MakeLoggerObject(id uuid.UUID) (*LogObject, error) {

	log := new(LogObject)
	path := fmt.Sprintf("%s%s_log.txt", os.Getenv("LOGPATH"), id.String())
	file, err := os.Create(path)

	if err != nil {
		file.Close()
		return &LogObject{}, err
	}

	log.File = file
	log.LogHandler = slog.NewTextHandler(log.File, nil)
	log.Path = path

	return log, nil
}

func (l *LogObject) CloseLogFile() error {
	err := l.File.Close()
	if err != nil {
		return err
	}
	return nil

}

func GetLog(filepath string) (string, error) {

	dir, err := os.ReadDir(os.Getenv("LOGPATH"))

	for _, fle := range dir {
		if fle.Name() == filepath {
			file, err := os.ReadFile(fmt.Sprintf("%s/%s", os.Getenv("LOGPATH"), fle.Name()))
			if err != nil {
				return "", err
			}

			return string(file), nil
		}
	}

	return "", err
}
