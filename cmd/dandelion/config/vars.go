package config

import (
	"github.com/jmoiron/sqlx"
	"github.com/tengattack/dandelion/mq"
	"github.com/tengattack/dandelion/repository"
)

var (
	// Conf is the main config
	Conf Config
	// Repo is git repository
	Repo *repository.Repository
	// DB is Database
	DB *sqlx.DB
	// MQ is MessageQueue
	MQ *mq.MessageQueue
)
