package handler

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"

	"danbooru-prompt-builder/config"
)

func validPathComponent(name string) bool {
	return name != "" && !strings.Contains(name, "..") && !strings.Contains(name, "/") && !strings.Contains(name, "\\")
}

func handleComfyWorkflows(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if name := r.URL.Query().Get("name"); name != "" {
			if !validPathComponent(name) {
				jsonError(w, "invalid workflow name", http.StatusBadRequest)
				return
			}
			wfPath := filepath.Join(cfg.WorkflowsPath, name+".json")
			data, err := os.ReadFile(wfPath)
			if err != nil {
				jsonError(w, "workflow not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		entries, err := os.ReadDir(cfg.WorkflowsPath)
		if err != nil {
			json.NewEncoder(w).Encode([]map[string]string{})
			return
		}
		var workflows []map[string]string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				name := strings.TrimSuffix(e.Name(), ".json")
				workflows = append(workflows, map[string]string{"name": name, "label": name})
			}
		}
		if workflows == nil {
			workflows = []map[string]string{}
		}
		json.NewEncoder(w).Encode(workflows)
	}
}

func handleComfyGenerate(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ClientID string            `json:"client_id"`
			Workflow string            `json:"workflow"`
			Macros   map[string]string `json:"macros"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Workflow == "" || req.Macros == nil {
			jsonError(w, "workflow and macros required", http.StatusBadRequest)
			return
		}
		if !validPathComponent(req.Workflow) {
			jsonError(w, "invalid workflow name", http.StatusBadRequest)
			return
		}
		wfPath := filepath.Join(cfg.WorkflowsPath, req.Workflow+".json")
		data, err := os.ReadFile(wfPath)
		if err != nil {
			jsonError(w, "workflow not found", http.StatusNotFound)
			return
		}
		content := string(data)
		for key, val := range req.Macros {
			jsonVal, _ := json.Marshal(val)
			content = strings.ReplaceAll(content, `"%`+key+`%"`, string(jsonVal))
			content = strings.ReplaceAll(content, `%`+key+`%`, val)
		}
		var promptData interface{}
		if err := json.Unmarshal([]byte(content), &promptData); err != nil {
			jsonError(w, "invalid workflow after macro replacement", http.StatusBadRequest)
			return
		}
		payload := map[string]interface{}{
			"client_id": req.ClientID,
			"prompt":    promptData,
		}
		comfyAddr := cfg.ComfyAddress
		body, _ := json.Marshal(payload)
		resp, err := http.Post(comfyAddr+"/prompt", "application/json", bytes.NewReader(body))
		if err != nil {
			jsonError(w, "comfyui connection failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		result, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			w.Write(result)
			return
		}
		var comfyResp struct {
			PromptID string `json:"prompt_id"`
			Error    string `json:"error"`
		}
		json.Unmarshal(result, &comfyResp)
		if comfyResp.Error != "" {
			jsonError(w, "comfyui error: "+comfyResp.Error, http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"prompt_id": comfyResp.PromptID})
	}
}

func handleComfyImage(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("filename")
		if filename == "" || !validPathComponent(filename) {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}

		if cfg.SavePath != "" {
			localPath := filepath.Join(cfg.SavePath, filename)
			if f, err := os.Open(localPath); err == nil {
				defer f.Close()
				stat, err := f.Stat()
				if err == nil && stat.Size() > 0 {
					w.Header().Set("Content-Type", "image/png")
					w.Header().Set("Cache-Control", "max-age=86400")
					http.ServeContent(w, r, filename, stat.ModTime(), f)
					return
				}
			}
		}

		viewURL := cfg.ComfyAddress + "/view?filename=" + url.QueryEscape(filename)
		if subfolder := r.URL.Query().Get("subfolder"); subfolder != "" {
			viewURL += "&subfolder=" + subfolder
		}
		if imgType := r.URL.Query().Get("type"); imgType != "" {
			viewURL += "&type=" + imgType
		}
		resp, err := http.Get(viewURL)
		if err != nil {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=86400")
		io.Copy(w, resp.Body)
	}
}

func readPNGPrompt(data []byte) (string, error) {
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		return "", nil
	}
	pos := 8
	for pos+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		chunkType := string(data[pos+4 : pos+8])
		if pos+12+length > len(data) {
			break
		}
		chunkData := data[pos+8 : pos+8+length]
		if chunkType == "tEXt" || chunkType == "iTXt" {
			nullIdx := bytes.IndexByte(chunkData, 0)
			if nullIdx > 0 && nullIdx < len(chunkData) {
				keyword := string(chunkData[:nullIdx])
				textData := chunkData[nullIdx+1:]
				if keyword == "prompt" {
					return string(textData), nil
				}
			}
		}
		pos += 12 + length
	}
	return "", nil
}

func handleComfyPromptInfo(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("filename")
		if filename == "" || !validPathComponent(filename) {
			jsonError(w, "filename required", http.StatusBadRequest)
			return
		}

		var pngData []byte
		if cfg.SavePath != "" {
			localPath := filepath.Join(cfg.SavePath, filename)
			if d, err := os.ReadFile(localPath); err == nil && len(d) > 0 {
				pngData = d
			}
		}

		if pngData == nil {
			viewURL := cfg.ComfyAddress + "/view?filename=" + url.QueryEscape(filename)
			if subfolder := r.URL.Query().Get("subfolder"); subfolder != "" {
				viewURL += "&subfolder=" + subfolder
			}
			if imgType := r.URL.Query().Get("type"); imgType != "" {
				viewURL += "&type=" + imgType
			}
			resp, err := http.Get(viewURL)
			if err != nil {
				jsonError(w, "comfyui request failed: "+err.Error(), http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()
			pngData, err = io.ReadAll(resp.Body)
			if err != nil {
				jsonError(w, "failed to read response", http.StatusBadGateway)
				return
			}
		}

		promptStr, err := readPNGPrompt(pngData)
		if err != nil || promptStr == "" {
			jsonError(w, "prompt not found in PNG", http.StatusNotFound)
			return
		}

		var promptJSON interface{}
		if err := json.Unmarshal([]byte(promptStr), &promptJSON); err != nil {
			jsonError(w, "invalid prompt JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"prompt": promptJSON})
	}
}

func handleComfyObjectInfo(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeType := strings.TrimPrefix(r.URL.Path, "/api/comfy/object_info/")
		if nodeType == "" {
			jsonError(w, "node type required", http.StatusBadRequest)
			return
		}
		comfyAddr := cfg.ComfyAddress
		resp, err := http.Get(comfyAddr + "/object_info/" + nodeType)
		if err != nil {
			jsonError(w, "comfyui request failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func handleComfySaveImage(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.Filename == "" || !validPathComponent(req.Filename) {
			jsonError(w, "filename required", http.StatusBadRequest)
			return
		}
		viewURL := cfg.ComfyAddress + "/view?filename=" + url.QueryEscape(req.Filename)
		if req.Subfolder != "" {
			viewURL += "&subfolder=" + req.Subfolder
		}
		if req.Type != "" {
			viewURL += "&type=" + req.Type
		}
		resp, err := http.Get(viewURL)
		if err != nil {
			jsonError(w, "failed to fetch image from comfyui: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		saveDir := cfg.SavePath
		if saveDir == "" {
			saveDir = "."
		}
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			jsonError(w, "failed to create save directory: "+err.Error(), http.StatusInternalServerError)
			return
		}
		savePath := filepath.Join(saveDir, req.Filename)
		outFile, err := os.Create(savePath)
		if err != nil {
			jsonError(w, "failed to create file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer outFile.Close()
		if _, err := io.Copy(outFile, resp.Body); err != nil {
			jsonError(w, "failed to save image: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": savePath})
	}
}

func handleComfyScanHistory(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		savePath := cfg.SavePath
		if savePath == "" {
			savePath = "./output"
		}
		entries, err := os.ReadDir(savePath)
		if err != nil {
			jsonError(w, "failed to read save path: "+err.Error(), http.StatusInternalServerError)
			return
		}
		var files []string
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(savePath, entry.Name()))
			if err != nil {
				continue
			}
			promptStr, err := readPNGPrompt(data)
			if err != nil || promptStr == "" {
				continue
			}
			files = append(files, entry.Name())
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"files": files})
	}
}

func handleComfyWS(cfg *config.Config) http.HandlerFunc {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("clientId")
		if clientID == "" {
			http.Error(w, "clientId required", http.StatusBadRequest)
			return
		}

		browserConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer browserConn.Close()

		u := url.URL{Scheme: "ws", Host: strings.TrimPrefix(cfg.ComfyAddress, "http://"), Path: "/ws", RawQuery: "clientId=" + url.QueryEscape(clientID)}
		comfyConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return
		}
		defer comfyConn.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			defer cancel()
			for {
				mt, msg, err := comfyConn.ReadMessage()
				if err != nil {
					return
				}
				if err := browserConn.WriteMessage(mt, msg); err != nil {
					return
				}
			}
		}()
		go func() {
			defer cancel()
			for {
				_, msg, err := browserConn.ReadMessage()
				if err != nil {
					return
				}
				if err := comfyConn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()
		<-ctx.Done()
	}
}
