package util

import "time"

func timeNowDate() string {
	return time.Now().Format("20060102")
}

// NowTime 返回当前时间。
func NowTime() time.Time { return time.Now() }

// TodayDate 返回 Asia/Shanghai 时区的今天日期（与 DB 会话 TimeZone=Asia/Shanghai 一致）。
func TodayDate() string {
	return time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
}
