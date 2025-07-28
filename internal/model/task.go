package model

import (
	"time"
)

// TaskStatus 表示下载任务的状态
type TaskStatus string

const (
	TaskStatusDownloading TaskStatus = "downloading" // 下载中
	TaskStatusCompleted   TaskStatus = "completed"   // 已完成
	TaskStatusError       TaskStatus = "error"       // 出错
)

// Task 表示一个下载任务
type Task struct {
	ID           string     `json:"id"`            // 任务ID
	URL          string     `json:"url"`           // 直播流URL
	FileName     string     `json:"file_name"`     // 文件名
	FilePath     string     `json:"file_path"`     // 文件路径
	Status       TaskStatus `json:"status"`        // 任务状态
	FileSize     int64      `json:"file_size"`     // 文件大小（字节）
	StartTime    time.Time  `json:"start_time"`    // 开始时间
	EndTime      *time.Time `json:"end_time"`     // 结束时间
	ErrorMessage string     `json:"error_message"` // 错误信息
}