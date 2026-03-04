package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

func cmdWeb() *cli.Command {
	return &cli.Command{
		Name:  "web",
		Usage: "启动 Web 管理页面",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "listen", Value: "127.0.0.1:8090", Usage: "监听地址，如 127.0.0.1:8090"},
		},
		Action: func(ctx *cli.Context) error {
			listen := strings.TrimSpace(ctx.String("listen"))
			if listen == "" {
				listen = "127.0.0.1:8090"
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/", serveWebIndex)
			mux.HandleFunc("/api/run", handleWebRun)
			mux.HandleFunc("/api/run-stream", handleWebRunStream)
			mux.HandleFunc("/api/files", handleWebFiles)
			mux.HandleFunc("/api/upload", handleWebUpload)
			mux.HandleFunc("/api/download", handleWebDownload)

			fmt.Printf("Web 页面已启动: http://%s\n", listen)
			fmt.Println("说明: 页面执行的是当前 safeline CLI 命令，覆盖 README 中全部能力。")
			return http.ListenAndServe(listen, mux)
		},
	}
}

type webRunRequest struct {
	Config      string `json:"config"`
	Addr        string `json:"addr"`
	Token       string `json:"token"`
	Match       string `json:"match"`
	Mode        string `json:"mode"`
	Timeout     int64  `json:"timeout"`
	Debug       bool   `json:"debug"`
	Command     string `json:"command"`
	ExecTimeout int64  `json:"exec_timeout"`
}

type webRunResponse struct {
	Args       []string `json:"args"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	ExitCode   int      `json:"exit_code"`
	DurationMS int64    `json:"duration_ms"`
	Error      string   `json:"error,omitempty"`
}

type webFileItem struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

type webRunStreamEvent struct {
	Type       string   `json:"type"`
	Args       []string `json:"args,omitempty"`
	Stream     string   `json:"stream,omitempty"`
	Text       string   `json:"text,omitempty"`
	ExitCode   int      `json:"exit_code,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func serveWebIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, webIndexHTML)
}

func handleWebRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req webRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, webRunResponse{Error: "invalid json: " + err.Error()})
		return
	}

	args, err := buildArgsFromRunRequest(req)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, webRunResponse{Error: err.Error()})
		return
	}

	execTimeout := req.ExecTimeout
	if execTimeout <= 0 {
		execTimeout = 300
	}

	exe, err := os.Executable()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, webRunResponse{Error: "获取可执行文件路径失败: " + err.Error()})
		return
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(execTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	resp := webRunResponse{
		Args:       args,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMS: time.Since(start).Milliseconds(),
	}
	if ctx.Err() == context.DeadlineExceeded {
		resp.ExitCode = -1
		resp.Error = fmt.Sprintf("执行超时（%ds）", execTimeout)
		respondJSON(w, http.StatusRequestTimeout, resp)
		return
	}
	if err != nil {
		resp.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			resp.ExitCode = -1
		}
	}

	respondJSON(w, http.StatusOK, resp)
}

func handleWebRunStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	var req webRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	args, err := buildArgsFromRunRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	execTimeout := req.ExecTimeout
	if execTimeout <= 0 {
		execTimeout = 300
	}
	exe, err := os.Executable()
	if err != nil {
		http.Error(w, "获取可执行文件路径失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(execTimeout)*time.Second)
	defer cancel()
	start := time.Now()

	cmd := exec.CommandContext(ctx, exe, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = writeStreamEvent(w, flusher, &sync.Mutex{}, webRunStreamEvent{Type: "exit", ExitCode: -1, Error: err.Error()})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = writeStreamEvent(w, flusher, &sync.Mutex{}, webRunStreamEvent{Type: "exit", ExitCode: -1, Error: err.Error()})
		return
	}
	if err = cmd.Start(); err != nil {
		_ = writeStreamEvent(w, flusher, &sync.Mutex{}, webRunStreamEvent{Type: "exit", ExitCode: -1, Error: err.Error()})
		return
	}

	mu := &sync.Mutex{}
	_ = writeStreamEvent(w, flusher, mu, webRunStreamEvent{Type: "start", Args: args})

	wg := sync.WaitGroup{}
	wg.Add(2)
	streamReader := func(name string, rd io.ReadCloser) {
		defer wg.Done()
		defer rd.Close()
		buf := make([]byte, 1024)
		reader := bufio.NewReader(rd)
		for {
			n, e := reader.Read(buf)
			if n > 0 {
				_ = writeStreamEvent(w, flusher, mu, webRunStreamEvent{Type: "chunk", Stream: name, Text: string(buf[:n])})
			}
			if e != nil {
				break
			}
		}
	}
	go streamReader("stdout", stdout)
	go streamReader("stderr", stderr)
	wg.Wait()

	waitErr := cmd.Wait()
	exitCode := 0
	errText := ""
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = -1
		errText = fmt.Sprintf("执行超时（%ds）", execTimeout)
	} else if waitErr != nil {
		errText = waitErr.Error()
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	_ = writeStreamEvent(w, flusher, mu, webRunStreamEvent{
		Type:       "exit",
		ExitCode:   exitCode,
		DurationMS: time.Since(start).Milliseconds(),
		Error:      errText,
	})
}

func buildArgsFromRunRequest(req webRunRequest) ([]string, error) {
	cmdTokens, err := splitShellArgs(req.Command)
	if err != nil {
		return nil, fmt.Errorf("解析命令失败: %s", err.Error())
	}
	if len(cmdTokens) == 0 {
		return nil, fmt.Errorf("命令不能为空")
	}
	if cmdTokens[0] == "web" {
		return nil, fmt.Errorf("不允许在页面里递归执行 web 命令")
	}

	args := make([]string, 0, len(cmdTokens)+16)
	if strings.TrimSpace(req.Config) != "" {
		args = append(args, "--config", strings.TrimSpace(req.Config))
	}
	if strings.TrimSpace(req.Addr) != "" {
		args = append(args, "--addr", strings.TrimSpace(req.Addr))
	}
	if strings.TrimSpace(req.Token) != "" {
		args = append(args, "--token", strings.TrimSpace(req.Token))
	}
	if strings.TrimSpace(req.Match) != "" {
		args = append(args, "--match", strings.TrimSpace(req.Match))
	}
	if req.Timeout > 0 {
		args = append(args, "--timeout", strconv.FormatInt(req.Timeout, 10))
	}
	if req.Debug {
		args = append(args, "--debug")
	}
	finalCmdTokens := cmdTokens
	if len(cmdTokens) > 0 && cmdTokens[0] == "website" && !hasModeFlag(cmdTokens) && strings.TrimSpace(req.Mode) != "" {
		injected := make([]string, 0, len(cmdTokens)+2)
		injected = append(injected, cmdTokens[0], "--mode", strings.TrimSpace(req.Mode))
		if len(cmdTokens) > 1 {
			injected = append(injected, cmdTokens[1:]...)
		}
		finalCmdTokens = injected
	}
	args = append(args, finalCmdTokens...)
	return args, nil
}

func writeStreamEvent(w http.ResponseWriter, flusher http.Flusher, mu *sync.Mutex, evt webRunStreamEvent) error {
	mu.Lock()
	defer mu.Unlock()
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	if _, err = w.Write(append(b, '\n')); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func handleWebFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files := make([]webFileItem, 0, 32)
	appendFiles := func(baseDir string, relPrefix string) {
		entries, err := ioutil.ReadDir(baseDir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !(strings.HasSuffix(name, ".csv") || strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".txt")) {
				continue
			}
			displayName := name
			if relPrefix != "" {
				displayName = filepath.ToSlash(filepath.Join(relPrefix, name))
			}
			files = append(files, webFileItem{
				Name:    displayName,
				Size:    entry.Size(),
				ModTime: entry.ModTime().Format(time.RFC3339),
			})
		}
	}

	appendFiles(".", "")
	appendFiles("build/templates", "build/templates")
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	respondJSON(w, http.StatusOK, files)
}

func handleWebUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "parse form failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	f, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	name := filepath.Base(header.Filename)
	if name == "." || name == "" {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	dstPath := filepath.Join(".", name)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, "save file failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, f); err != nil {
		http.Error(w, "write file failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "file": name})
}

func handleWebDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	path, err := resolveDownloadPath(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func resolveDownloadPath(name string) (string, error) {
	clean := filepath.Clean(name)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid file path")
	}

	cand := []string{
		filepath.Join(".", clean),
		filepath.Join("build", "templates", clean),
	}
	for _, p := range cand {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("file not found")
}

func respondJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func splitShellArgs(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	out := make([]string, 0, 16)
	var buf bytes.Buffer
	quote := byte(0)
	escaped := false
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, buf.String())
			buf.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			buf.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			buf.WriteByte(ch)
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\n' {
			flush()
			continue
		}
		buf.WriteByte(ch)
	}
	if escaped {
		return nil, fmt.Errorf("命令以转义符结尾")
	}
	if quote != 0 {
		return nil, fmt.Errorf("引号未闭合")
	}
	flush()
	return out, nil
}

func hasModeFlag(tokens []string) bool {
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "--mode" || tokens[i] == "-m" {
			return true
		}
	}
	return false
}

const webIndexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Safeline Web 控制台</title>
  <style>
    :root {
      --bg: #f4f6f8;
      --panel: #ffffff;
      --fg: #1f2933;
      --muted: #697586;
      --line: #d8dee4;
      --brand: #0b6bcb;
      --brand2: #0d4f9a;
      --ok: #057a55;
      --err: #b42318;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "PingFang SC", "Noto Sans SC", "Microsoft YaHei", sans-serif;
      color: var(--fg);
      background: radial-gradient(circle at 100% 0%, #e8f3ff, var(--bg) 42%);
    }
    .wrap {
      max-width: 1180px;
      margin: 20px auto;
      padding: 0 16px 24px;
    }
    .hero {
      background: linear-gradient(130deg, #0d4f9a, #0b6bcb);
      color: #fff;
      border-radius: 14px;
      padding: 16px 18px;
      box-shadow: 0 10px 26px rgba(13, 79, 154, 0.25);
      margin-bottom: 14px;
    }
    .hero h1 { margin: 0 0 6px; font-size: 22px; }
    .hero p { margin: 0; opacity: 0.9; font-size: 13px; }
    .grid {
      display: grid;
      grid-template-columns: 1.2fr 1fr;
      gap: 12px;
    }
    .card {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      box-shadow: 0 3px 8px rgba(10, 24, 40, 0.03);
    }
    h2 { margin: 4px 0 10px; font-size: 15px; }
    label {
      display: block;
      font-size: 12px;
      color: var(--muted);
      margin: 7px 0 4px;
    }
    input, textarea, select {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 8px 10px;
      font-size: 13px;
      background: #fff;
    }
    textarea { min-height: 82px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
    .row {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 10px;
    }
    .row3 {
      display: grid;
      grid-template-columns: 1fr 1fr 1fr;
      gap: 10px;
    }
    .btn {
      border: 1px solid transparent;
      background: var(--brand);
      color: #fff;
      border-radius: 9px;
      padding: 8px 12px;
      cursor: pointer;
      font-size: 13px;
      margin-right: 6px;
      margin-top: 8px;
    }
    .btn:hover { background: var(--brand2); }
    .btn.sub {
      background: #fff;
      color: var(--brand2);
      border-color: var(--brand2);
    }
    .chips {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      margin-bottom: 8px;
    }
    .chip {
      font-size: 12px;
      background: #edf5ff;
      color: #0d4f9a;
      border: 1px solid #cddff5;
      border-radius: 999px;
      padding: 4px 10px;
      cursor: pointer;
    }
    pre {
      margin: 8px 0 0;
      background: #0d1117;
      color: #dbe7ff;
      border-radius: 10px;
      padding: 10px;
      min-height: 180px;
      max-height: 440px;
      overflow: auto;
      font-size: 12px;
      line-height: 1.5;
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    }
    .status { font-size: 12px; margin-top: 8px; }
    .ok { color: var(--ok); }
    .err { color: var(--err); }
    .files {
      margin-top: 6px;
      border: 1px solid var(--line);
      border-radius: 10px;
      overflow: hidden;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
    }
    th, td {
      border-bottom: 1px solid #edf1f5;
      text-align: left;
      padding: 6px 8px;
    }
    th { background: #f8fafc; }
    .small { font-size: 12px; color: var(--muted); }
    @media (max-width: 960px) {
      .grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
<div class="wrap">
  <div class="hero">
    <h1>Safeline Web 控制台</h1>
    <p>页面会将操作转换为原始 safeline CLI 命令执行，覆盖 README 中的所有能力。</p>
  </div>

  <div class="grid">
    <div class="card">
      <h2>全局参数</h2>
      <div class="row">
        <div>
          <label>config（默认 token.csv）</label>
          <input id="config" value="token.csv" />
        </div>
        <div>
          <label>match（可选）</label>
          <input id="match" placeholder="如 169.254.0.4" />
        </div>
      </div>
      <div class="row">
        <div>
          <label>addr（与 token 搭配，优先于 config）</label>
          <input id="addr" placeholder="https://169.254.0.4:1443" />
        </div>
        <div>
          <label>token（与 addr 搭配）</label>
          <input id="token" placeholder="API-TOKEN" />
        </div>
      </div>
      <div class="row3">
        <div>
          <label>请求超时(s)</label>
          <input id="timeout" type="number" value="5" />
        </div>
        <div>
          <label>执行超时(s)</label>
          <input id="exec_timeout" type="number" value="300" />
        </div>
        <div>
          <label>debug</label>
          <select id="debug"><option value="false">false</option><option value="true">true</option></select>
        </div>
      </div>
      <div class="row">
        <div>
          <label>当前模式（website 命令）</label>
          <select id="mode">
            <option value="HardwareReverseProxy">HardwareReverseProxy</option>
            <option value="SoftwareReverseProxy">SoftwareReverseProxy</option>
            <option value="TransparentProxy">TransparentProxy</option>
            <option value="TransparentBridge">TransparentBridge</option>
          </select>
        </div>
        <div>
          <label>模式应用规则</label>
          <input value="website 命令未显式传 --mode 时，自动使用这里的值" disabled />
        </div>
      </div>

      <h2>命令构建器</h2>
      <label>获取信息</label>
      <div class="chips">
        <button class="chip" data-cmd="website ls">站点列表</button>
        <button class="chip" data-cmd="policy ls">防护策略列表</button>
        <button class="chip" data-cmd="cert ls">证书列表</button>
        <button class="chip" data-cmd="assetgroup ls">资产组列表</button>
        <button class="chip" data-cmd="ipgroup ls">IP组列表</button>
        <button class="chip" data-cmd="ipgroup getip --byid 1">IP组查询IP</button>
        <button class="chip" data-cmd="ipgroup allip">IP组联合查询</button>
        <button class="chip" data-cmd="acl ls">ACL列表</button>
        <button class="chip" data-cmd="acl getip --byid 1">ACL查询IP</button>
        <button class="chip" data-cmd="global-rule ls">全局规则列表</button>
        <button class="chip" data-cmd="custom-rule ls">站点规则列表</button>
      </div>

      <label>更新信息</label>
      <div class="chips">
        <button class="chip" data-cmd="website create -f template.csv">创建站点</button>
        <button class="chip" data-cmd="website update -f template.csv -s \"0,1,2\"">更新站点</button>
        <button class="chip" data-cmd="ipgroup addip --byid 1 1.2.3.4">IP组添加IP</button>
        <button class="chip" data-cmd="ipgroup delip --byid 1 1.2.3.4">IP组删除IP</button>
        <button class="chip" data-cmd="ipgroup create --name test --comment test 1.1.1.1/32">创建IP组</button>
        <button class="chip" data-cmd="ipgroup delete 1">删除IP组</button>
        <button class="chip" data-cmd="acl addip --byid 1 1.2.3.4">ACL添加IP</button>
        <button class="chip" data-cmd="acl delip --byid 1 1.2.3.4">ACL删除IP</button>
        <button class="chip" data-cmd="acl clearip --byid 1">ACL清空IP</button>
        <button class="chip" data-cmd="acl create --name test --expire 0 1.1.1.1/32">创建ACL规则</button>
        <button class="chip" data-cmd="acl delete 1">删除ACL规则</button>
      </div>

      <label>导出/导入信息</label>
      <div class="chips">
        <button class="chip" data-cmd="website export -f template">导出站点模板</button>
        <button class="chip" data-cmd="ipgroup exportip -f ipgroup_ips --format csv">导出IP组IP</button>
        <button class="chip" data-cmd="global-rule export -f global_rules">导出全局规则</button>
        <button class="chip" data-cmd="global-rule import -f global_rules.json">导入全局规则</button>
        <button class="chip" data-cmd="custom-rule export -f custom_rules">导出站点规则</button>
        <button class="chip" data-cmd="custom-rule import -f custom_rules.json">导入站点规则</button>
      </div>

      <label>命令（不含可执行文件名，例如：website ls）</label>
      <textarea id="command" placeholder="输入任意 safeline 命令，例如 website ls"></textarea>
      <div>
        <button id="run" class="btn">执行命令</button>
        <button id="clear" class="btn sub">清空输出</button>
        <button id="clearCache" class="btn sub">清除缓存</button>
      </div>
      <div id="status" class="status small">等待执行</div>
      <pre id="output">$ 输出会显示在这里</pre>
    </div>

    <div class="card">
      <h2>文件管理（模板/导入导出）</h2>
      <p class="small">可上传 CSV/JSON 文件；导出的文件可在列表里直接下载。系统也会展示 build/templates 下的模板。</p>

      <form id="uploadForm">
        <label>上传文件</label>
        <input type="file" id="file" name="file" />
        <button class="btn" type="submit">上传</button>
        <button class="btn sub" id="refreshFiles" type="button">刷新列表</button>
      </form>

      <div class="files">
        <table>
          <thead>
          <tr><th>文件</th><th>大小</th><th>修改时间</th></tr>
          </thead>
          <tbody id="fileRows"></tbody>
        </table>
      </div>
    </div>
  </div>
</div>

<script>
const output = document.getElementById('output');
const statusEl = document.getElementById('status');
const STORAGE_KEY = 'safeline_web_global_config_v1';
const CONFIG_FIELDS = ['config', 'addr', 'token', 'match', 'mode', 'timeout', 'exec_timeout', 'debug', 'command'];

function esc(text) {
  return text
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function setStatus(msg, ok=true) {
  statusEl.textContent = msg;
  statusEl.className = 'status ' + (ok ? 'ok' : 'err');
}

function currentFormState() {
  return {
    config: document.getElementById('config').value.trim(),
    addr: document.getElementById('addr').value.trim(),
    token: document.getElementById('token').value.trim(),
    match: document.getElementById('match').value.trim(),
    mode: document.getElementById('mode').value,
    timeout: document.getElementById('timeout').value || '5',
    exec_timeout: document.getElementById('exec_timeout').value || '300',
    debug: document.getElementById('debug').value || 'false',
    command: document.getElementById('command').value || '',
  };
}

function saveConfigToStorage() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(currentFormState()));
  } catch (e) {}
}

function loadConfigFromStorage() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const data = JSON.parse(raw);
    for (const key of CONFIG_FIELDS) {
      if (!(key in data)) continue;
      const el = document.getElementById(key);
      if (!el) continue;
      el.value = String(data[key]);
    }
  } catch (e) {}
}

function clearConfigStorage() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch (e) {}
}

async function runCmd() {
  saveConfigToStorage();
  const req = {
    config: document.getElementById('config').value.trim(),
    addr: document.getElementById('addr').value.trim(),
    token: document.getElementById('token').value.trim(),
    match: document.getElementById('match').value.trim(),
    mode: document.getElementById('mode').value,
    timeout: parseInt(document.getElementById('timeout').value || '5', 10),
    exec_timeout: parseInt(document.getElementById('exec_timeout').value || '300', 10),
    debug: document.getElementById('debug').value === 'true',
    command: document.getElementById('command').value.trim(),
  };
  if (!req.command) {
    setStatus('命令不能为空', false);
    return;
  }

  setStatus('执行中...');
  output.textContent = '$ safeline ' + req.command + '\n';
  const appendOutput = (txt) => {
    output.textContent += txt;
    output.scrollTop = output.scrollHeight;
  };
  try {
    const res = await fetch('/api/run-stream', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(req),
    });
    if (!res.ok || !res.body) {
      const txt = await res.text();
      throw new Error(txt || ('HTTP ' + res.status));
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let finalEvent = null;
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';
      for (const line of lines) {
        if (!line.trim()) continue;
        const evt = JSON.parse(line);
        if (evt.type === 'start') {
          appendOutput('$ args: ' + (evt.args || []).join(' ') + '\n');
          continue;
        }
        if (evt.type === 'chunk') {
          appendOutput(evt.text || '');
          continue;
        }
        if (evt.type === 'exit') {
          finalEvent = evt;
        }
      }
    }
    if (buffer.trim()) {
      const evt = JSON.parse(buffer);
      if (evt.type === 'exit') finalEvent = evt;
    }

    if (finalEvent) {
      appendOutput('\n$ duration_ms: ' + (finalEvent.duration_ms || 0) + '\n');
      if (finalEvent.error) appendOutput('$ error: ' + finalEvent.error + '\n');
      appendOutput('$ exit_code: ' + (finalEvent.exit_code ?? 0) + '\n');
      setStatus(finalEvent.error ? '执行完成（有错误）' : '执行完成', !finalEvent.error);
    } else {
      setStatus('执行完成（未收到结束事件）', false);
    }
    await loadFiles();
  } catch (err) {
    appendOutput('\n请求失败: ' + err + '\n');
    setStatus('请求失败', false);
  }
}

async function loadFiles() {
  const rows = document.getElementById('fileRows');
  rows.innerHTML = '<tr><td colspan="3">加载中...</td></tr>';
  try {
    const res = await fetch('/api/files');
    const data = await res.json();
    if (!Array.isArray(data) || data.length === 0) {
      rows.innerHTML = '<tr><td colspan="3">暂无文件</td></tr>';
      return;
    }
    rows.innerHTML = data.map(item => {
      const link = '<a href="/api/download?name=' + encodeURIComponent(item.name) + '">' + esc(item.name) + '</a>';
      return '<tr><td>' + link + '</td><td>' + item.size + '</td><td>' + esc(item.mod_time) + '</td></tr>';
    }).join('');
  } catch (err) {
    rows.innerHTML = '<tr><td colspan="3">加载失败: ' + esc(String(err)) + '</td></tr>';
  }
}

async function uploadFile(e) {
  e.preventDefault();
  const input = document.getElementById('file');
  if (!input.files || input.files.length === 0) {
    setStatus('请选择文件后再上传', false);
    return;
  }
  const fd = new FormData();
  fd.append('file', input.files[0]);
  try {
    const res = await fetch('/api/upload', { method: 'POST', body: fd });
    if (!res.ok) {
      const txt = await res.text();
      throw new Error(txt);
    }
    setStatus('上传成功');
    input.value = '';
    await loadFiles();
  } catch (err) {
    setStatus('上传失败: ' + err, false);
  }
}

document.getElementById('run').addEventListener('click', runCmd);
document.getElementById('clear').addEventListener('click', () => {
  output.textContent = '$ 输出会显示在这里';
  setStatus('已清空输出');
});
document.getElementById('clearCache').addEventListener('click', () => {
  clearConfigStorage();
  document.getElementById('config').value = 'token.csv';
  document.getElementById('addr').value = '';
  document.getElementById('token').value = '';
  document.getElementById('match').value = '';
  document.getElementById('mode').value = 'HardwareReverseProxy';
  document.getElementById('timeout').value = '5';
  document.getElementById('exec_timeout').value = '300';
  document.getElementById('debug').value = 'false';
  document.getElementById('command').value = '';
  setStatus('已清除缓存并恢复默认值');
});
document.getElementById('uploadForm').addEventListener('submit', uploadFile);
document.getElementById('refreshFiles').addEventListener('click', loadFiles);
document.querySelectorAll('.chip').forEach(el => {
  el.addEventListener('click', () => {
    document.getElementById('command').value = el.dataset.cmd || '';
    saveConfigToStorage();
  });
});
for (const key of CONFIG_FIELDS) {
  const el = document.getElementById(key);
  if (!el) continue;
  el.addEventListener('change', saveConfigToStorage);
  el.addEventListener('input', saveConfigToStorage);
}
loadConfigFromStorage();
loadFiles();
</script>
</body>
</html>`
