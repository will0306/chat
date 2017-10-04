package model

import (
    "github.com/astaxie/beego/orm"
)

type User struct {
    Id          int
    Name        string
	Pwd			string
}
// 多字段唯一键
func (u *User) TableUnique() [][]string {
	return [][]string{
		[]string{"Name"},
	}
}


func init() {
    // 需要在init中注册定义的model
    orm.RegisterModel(new(User))
}
