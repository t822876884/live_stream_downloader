package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	_ "strconv"
	"strings" // 添加strings包
	_ "time"

	"github.com/luan78zao/live_stream_downloader/internal/downloader"
)

// Handler 处理HTTP请求
type Handler struct {
	downloader *downloader.Downloader
	templates  *template.Template
	dataDir    string
}

// New 创建一个新的处理器
func New(downloader *downloader.Downloader, templatesDir, dataDir string) (*Handler, error) {
	// 在New函数中，创建模板函数映射
	funcMap := template.FuncMap{
		"contains": strings.Contains,
		"formatSize": func(size int64) string {
			if size < 1024 {
				return fmt.Sprintf("%d 字节", size)
			} else if size < 1024*1024 {
				return fmt.Sprintf("%.2f KB", float64(size)/1024)
			} else if size < 1024*1024*1024 {
				return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
			} else {
				return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
			}
		},
	}

	// 将函数映射应用到模板
	//templates := template.Must(template.New("").Funcs(funcMap).ParseGlob(filepath.Join("web", "templates", "*.html")))
	// 使用绝对路径而不是相对路径
	templatesPath := filepath.Join("/app", "web", "templates", "*.html")
	templates := template.Must(template.New("").Funcs(funcMap).ParseGlob(templatesPath))

	return &Handler{
		downloader: downloader,
		templates:  templates,
		dataDir:    dataDir,
	}, nil
}

// RegisterRoutes 注册HTTP路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 静态文件
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// 数据文件（下载的视频）
	mux.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir(h.dataDir))))

	// 页面
	mux.HandleFunc("/", h.handleIndex)
	mux.HandleFunc("/active", h.handleActive)
	mux.HandleFunc("/completed", h.handleCompleted)

	// API
	mux.HandleFunc("/api/tasks", h.handleCreateTask)
	mux.HandleFunc("/api/tasks/active", h.handleGetActiveTasks)
	mux.HandleFunc("/api/tasks/completed", h.handleGetCompletedTasks)
	mux.HandleFunc("/api/tasks/stop/", h.handleStopTask)
	// 添加删除任务的路由
	mux.HandleFunc("/api/tasks/delete/active/", h.handleDeleteActiveTask)
	mux.HandleFunc("/api/tasks/delete/completed/", h.handleDeleteCompletedTask)
}

// handleIndex 处理首页请求
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	h.templates.ExecuteTemplate(w, "index.html", nil)
}

// handleActive 处理活动任务页面请求
func (h *Handler) handleActive(w http.ResponseWriter, r *http.Request) {
	activeTasks := h.downloader.GetActiveTasks()
	h.templates.ExecuteTemplate(w, "active.html", map[string]interface{}{
		"Tasks": activeTasks,
	})
}

// handleCompleted 处理已完成任务页面请求
func (h *Handler) handleCompleted(w http.ResponseWriter, r *http.Request) {
	completedTasks := h.downloader.GetCompletedTasks()
	h.templates.ExecuteTemplate(w, "completed.html", map[string]interface{}{
		"Tasks": completedTasks,
	})
}

// handleCreateTask 处理创建任务请求
func (h *Handler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 定义请求结构
	type CreateTaskRequest struct {
		URL      string `json:"url"`
		FileName string `json:"file_name"`
	}

	var url, fileName string

	// 根据Content-Type处理不同格式的请求
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		// 解析JSON请求体
		var req CreateTaskRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("解析JSON失败: %v", err), http.StatusBadRequest)
			return
		}
		url = req.URL
		fileName = req.FileName
	} else {
		// 解析表单数据（兼容现有的表单提交方式）
		if err := r.ParseForm(); err != nil {
			http.Error(w, "解析表单失败", http.StatusBadRequest)
			return
		}
		url = r.FormValue("url")
		fileName = r.FormValue("file_name")
	}

	// 验证URL
	if url == "" {
		http.Error(w, "URL不能为空", http.StatusBadRequest)
		return
	}

	// 创建任务
	task, err := h.downloader.CreateTask(url, fileName)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建任务失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回任务信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// handleGetActiveTasks 处理获取活动任务请求
func (h *Handler) handleGetActiveTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取活动任务
	activeTasks := h.downloader.GetActiveTasks()

	// 返回任务信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activeTasks)
}

// handleGetCompletedTasks 处理获取已完成任务请求
func (h *Handler) handleGetCompletedTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取已完成任务
	completedTasks := h.downloader.GetCompletedTasks()

	// 返回任务信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedTasks)
}

// handleStopTask 处理停止任务请求
func (h *Handler) handleStopTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取任务ID
	taskID := r.URL.Path[len("/api/tasks/stop/"):]
	if taskID == "" {
		http.Error(w, "任务ID不能为空", http.StatusBadRequest)
		return
	}

	// 停止任务
	if err := h.downloader.StopTask(taskID); err != nil {
		http.Error(w, fmt.Sprintf("停止任务失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回成功信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteActiveTask 处理删除活动任务请求
func (h *Handler) handleDeleteActiveTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取任务ID
	taskID := r.URL.Path[len("/api/tasks/delete/active/"):]
	if taskID == "" {
		http.Error(w, "任务ID不能为空", http.StatusBadRequest)
		return
	}

	// 删除任务
	if err := h.downloader.DeleteActiveTask(taskID); err != nil {
		http.Error(w, fmt.Sprintf("删除任务失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回成功信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteCompletedTask 处理删除已完成任务请求
func (h *Handler) handleDeleteCompletedTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取任务ID
	taskID := r.URL.Path[len("/api/tasks/delete/completed/"):]
	if taskID == "" {
		http.Error(w, "任务ID不能为空", http.StatusBadRequest)
		return
	}

	// 删除任务
	if err := h.downloader.DeleteCompletedTask(taskID); err != nil {
		http.Error(w, fmt.Sprintf("删除任务失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回成功信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
