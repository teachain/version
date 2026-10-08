package model

import "time"

type Version struct {
	ID            uint      `xorm:"pk autoincr 'id'"`
	ApplicationID uint      `xorm:"notnull unique(uk_app_tag) 'application_id'"`
	TagName       string    `xorm:"varchar(64) notnull unique(uk_app_tag) 'tag_name'"`
	Name          string    `xorm:"varchar(256) 'name'"`
	Body          string    `xorm:"text 'body'"`
	URL           string    `xorm:"varchar(512) 'url'"`
	PublishedAt   time.Time `xorm:"published_at"`
	CreatedAt     time.Time `xorm:"created"`
}

func (Version) TableName() string { return "version" }