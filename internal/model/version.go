package model

import "time"

type Version struct {
	ID            uint      `xorm:"pk autoincr 'id'" json:"id"`
	ApplicationID uint      `xorm:"notnull unique(uk_app_tag) 'application_id'" json:"application_id"`
	TagName       string    `xorm:"varchar(64) notnull unique(uk_app_tag) 'tag_name'" json:"tag_name"`
	Name          string    `xorm:"varchar(256) 'name'" json:"name"`
	Body          string    `xorm:"text 'body'" json:"body"`
	URL           string    `xorm:"varchar(512) 'url'" json:"url"`
	PublishedAt   time.Time `xorm:"published_at" json:"published_at"`
	CreatedAt     time.Time `xorm:"created" json:"created_at"`
}

func (Version) TableName() string { return "version" }