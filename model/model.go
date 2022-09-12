package model

import (
	"gorm.io/gorm"
)

var (
	//DB 数据库
	DB *gorm.DB
	//Err 数据库错误
	Err error
)

/*
*
对变量进行初始化
检查/修复程序的状态
注册
运行一次计算
*/
// func init() {
// 	dsn := "imgs:fr3W54XFs6KEzn5W@tcp(app.adtk.cn:3306)/imgs?charset=utf8mb4&parseTime=True&loc=Local"

// 	DB, Err = gorm.Open(mysql.Open(dsn), &gorm.Config{
// 		NamingStrategy: schema.NamingStrategy{
// 			TablePrefix:   "heng_", // 表名前缀，`User` 的表名应该是 `t_users`
// 			SingularTable: true,    // 使用单数表名，启用该选项，此时，`User` 的表名应该是 `t_user`
// 		},
// 	})

// 	if Err != nil {
// 		log.Println("数据库连接错误", Err)
// 		return
// 	}
// 	log.Println("数据库连接正常")
// 	// DB.AutoMigrate(&User{}) //自动迁移仅仅会创建表，缺少列和索引
// 	// DB.AutoMigrate(&File{})
// }
