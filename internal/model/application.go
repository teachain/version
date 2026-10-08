package model

import "time"

type Application struct {
	ID          uint       `json:"id" xorm:"pk autoincr 'id'"`
	Name        string     `json:"name" xorm:"varchar(128) notnull unique 'name'"`
	RepoURL     string     `json:"repo_url" xorm:"varchar(256) notnull 'repo_url'"`
	Enabled     bool       `json:"enabled" xorm:"notnull default true 'enabled'"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty" xorm:"last_check_at"`
	CreatedAt   time.Time  `json:"created_at" xorm:"created"`
	UpdatedAt   time.Time  `json:"updated_at" xorm:"updated"`
}

func (Application) TableName() string { return "application" }