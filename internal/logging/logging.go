package logging

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
)

type LogObject struct {
	File       *os.File
	LogHandler *slog.TextHandler
	Path       string
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
