package logger

import "github.com/sirupsen/logrus"

type Target interface {
	Fire(*logrus.Entry) error
	Levels() []logrus.Level
}
