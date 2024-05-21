package db

import "gorm.io/gorm"

type GenPictureInfo struct {
	gorm.Model
	Name      string   `gorm:"column:name" json:"name"`
	Prompt    string   `gorm:"column:prompt" json:"prompt"`
	Pictures  []string `gorm:"column:pictures; serializer:json" json:"pictures"`
	Status    int      `gorm:"column:status; default:0" json:"status"`
	Mutex     bool     `gorm:"column:mutex; default:false" json:"mutex"`
	CreatedBy string   `gorm:"column:created_by; default:nobody" json:"created_by"`
	Retry     int      `gorm:"column:retry; default:1" json:"retry"`
	Duration  int      `gorm:"column:duration" json:"duration"`
}

const tableNameGenPictureInfo = "gen_picture_infos"

func (g *GenPictureInfo) TableName() string {
	return tableNameGenPictureInfo
}
