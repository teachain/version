package repository

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

func NewEngine(dsn string) (*xorm.Engine, error) {
	eng, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("new engine: %w", err)
	}
	return eng, nil
}
