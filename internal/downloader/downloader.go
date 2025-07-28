package downloader

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/luan78zao/live_stream_downloader/internal/model"
)

// Downloader 管理直播流下载任务
type Downloader struct {
	mu             sync.RWMutex
	activeTasks    map[string]*model.Task
	completedTasks map[string]*model.Task
	dataDir        string
	clients        map[string]*http.Client
	cancelFuncs    map[string]context.CancelFunc
}

// New 创建一个新的下载器实例
func New(dataDir string) (*Downloader, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	return &Downloader{
		activeTasks:    make(map[string]*model.Task),
		completedTasks: make(map[string]*model.Task),
		dataDir:        dataDir,
		clients:        make(map[string]*http.Client),
		cancelFuncs:    make(map[string]context.CancelFunc),
	}, nil
}

// CreateTask 创建一个新的下载任务
func (d *Downloader) CreateTask(url, fileName string) (*model.Task, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 生成任务ID（使用时间戳）
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 确保文件名有效
	if fileName == "" {
		fileName = fmt.Sprintf("stream_%s.flv", taskID)
	} else if filepath.Ext(fileName) == "" {
		fileName = fileName + ".flv"
	}

	// 创建任务
	task := &model.Task{
		ID:        taskID,
		URL:       url,
		FileName:  fileName,
		FilePath:  filepath.Join(d.dataDir, fileName),
		Status:    model.TaskStatusDownloading,
		FileSize:  0,
		StartTime: time.Now(),
	}

	// 保存任务
	d.activeTasks[taskID] = task

	// 启动下载
	go d.startDownload(task)

	return task, nil
}

// startDownload 开始下载任务
func (d *Downloader) startDownload(task *model.Task) {
	// 创建上下文，以便能够取消下载
	ctx, cancel := context.WithCancel(context.Background())

	// 保存取消函数
	d.mu.Lock()
	d.cancelFuncs[task.ID] = cancel
	// 创建HTTP客户端，添加DNS解析重试
	client := &http.Client{
		Timeout: 0, // 不设置超时，因为是直播流
		Transport: &http.Transport{
			// 设置较长的DNS解析超时时间
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			// 增加最大空闲连接数
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	d.clients[task.ID] = client
	d.mu.Unlock()

	// 确保在函数退出时清理资源
	defer func() {
		d.mu.Lock()
		delete(d.cancelFuncs, task.ID)
		delete(d.clients, task.ID)
		d.mu.Unlock()
	}()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		d.handleDownloadError(task, fmt.Errorf("创建请求失败: %w", err))
		return
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		d.handleDownloadError(task, fmt.Errorf("发送请求失败: %w", err))
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		d.handleDownloadError(task, fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode))
		return
	}

	// 创建文件
	file, err := os.Create(task.FilePath)
	if err != nil {
		d.handleDownloadError(task, fmt.Errorf("创建文件失败: %w", err))
		return
	}
	defer file.Close()

	// 设置缓冲区大小
	bufSize := 32 * 1024 // 32KB
	buf := make([]byte, bufSize)

	// 开始下载
	var totalBytes int64
	updateInterval := time.Second * 1 // 每秒更新一次状态
	lastUpdate := time.Now()

	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			// 下载被取消
			return
		default:
			// 继续下载
		}

		// 读取数据
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// 写入文件
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				d.handleDownloadError(task, fmt.Errorf("写入文件失败: %w", writeErr))
				return
			}

			// 更新总字节数
			totalBytes += int64(n)

			// 定期更新任务状态
			if time.Since(lastUpdate) >= updateInterval {
				d.mu.Lock()
				task.FileSize = totalBytes
				d.mu.Unlock()
				lastUpdate = time.Now()
			}
		}

		// 检查是否到达流的末尾或发生错误
		if err != nil {
			if err == io.EOF {
				// 正常结束
				d.completeTask(task, totalBytes)
			} else {
				// 发生错误
				d.handleDownloadError(task, fmt.Errorf("读取数据失败: %w", err))
			}
			return
		}
	}
}

// handleDownloadError 处理下载错误
func (d *Downloader) handleDownloadError(task *model.Task, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 更新任务状态
	task.Status = model.TaskStatusError
	task.ErrorMessage = err.Error()
	endTime := time.Now()
	task.EndTime = &endTime

	// 将任务从活动任务移动到已完成任务
	delete(d.activeTasks, task.ID)
	d.completedTasks[task.ID] = task
}

// completeTask 完成下载任务
func (d *Downloader) completeTask(task *model.Task, totalBytes int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 更新任务状态
	task.Status = model.TaskStatusCompleted
	task.FileSize = totalBytes
	endTime := time.Now()
	task.EndTime = &endTime

	// 将任务从活动任务移动到已完成任务
	delete(d.activeTasks, task.ID)
	d.completedTasks[task.ID] = task
}

// StopTask 停止下载任务
func (d *Downloader) StopTask(taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查任务是否存在
	task, exists := d.activeTasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	// 取消下载
	if cancel, exists := d.cancelFuncs[taskID]; exists {
		cancel()
	}

	// 更新任务状态
	task.Status = model.TaskStatusCompleted
	endTime := time.Now()
	task.EndTime = &endTime

	// 将任务从活动任务移动到已完成任务
	delete(d.activeTasks, taskID)
	d.completedTasks[taskID] = task

	return nil
}

// GetActiveTasks 获取所有活动任务
func (d *Downloader) GetActiveTasks() []*model.Task {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tasks := make([]*model.Task, 0, len(d.activeTasks))
	for _, task := range d.activeTasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetCompletedTasks 获取所有已完成任务
func (d *Downloader) GetCompletedTasks() []*model.Task {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tasks := make([]*model.Task, 0, len(d.completedTasks))
	for _, task := range d.completedTasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTask 获取指定任务
func (d *Downloader) GetTask(taskID string) (*model.Task, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 先检查活动任务
	if task, exists := d.activeTasks[taskID]; exists {
		return task, true
	}

	// 再检查已完成任务
	if task, exists := d.completedTasks[taskID]; exists {
		return task, true
	}

	return nil, false
}

// DeleteActiveTask 删除正在下载的任务
func (d *Downloader) DeleteActiveTask(taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查任务是否存在
	task, exists := d.activeTasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	// 取消下载
	if cancel, exists := d.cancelFuncs[taskID]; exists {
		cancel()
	}

	// 删除文件
	if err := os.Remove(task.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	// 从活动任务中删除
	delete(d.activeTasks, taskID)

	return nil
}

// DeleteCompletedTask 删除已完成的任务
func (d *Downloader) DeleteCompletedTask(taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查任务是否存在
	task, exists := d.completedTasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	// 删除文件
	if err := os.Remove(task.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	// 从已完成任务中删除
	delete(d.completedTasks, taskID)

	return nil
}
