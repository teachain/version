package model

import "time"

type Application struct {
	ID          uint       `xorm:"pk autoincr 'id'"`
	Name        string     `xorm:"varchar(128) notnull unique 'name'"`
	RepoURL     string     `xorm:"varchar(256) notnull 'repo_url'"`
	Enabled     bool       `xorm:"notnull default true 'enabled'"`
	LastCheckAt *time.Time `xorm:"last_check_at"`
	CreatedAt   time.Time  `xorm:"created"`
	UpdatedAt   time.Time  `xorm:"updated"`
}

func (Application) TableName() string { return "application" }