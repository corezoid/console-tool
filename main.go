package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type request struct {
	Jsonrpc string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

type response struct {
	Jsonrpc string         `json:"jsonrpc"`
	ID      string         `json:"id,omitempty"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *respError     `json:"error,omitempty"`
}

type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	port := os.Getenv("GITCALL_PORT")
	if port == "" {
		log.Fatal("GITCALL_PORT env is required but not set")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", handler)

	fmt.Println("Listening on http://0.0.0.0:" + port)
	err := http.ListenAndServe("0.0.0.0:"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer check_panic(w, r)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		send_err(w, "", 1, err.Error())
		return
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		send_err(w, "", 1, err.Error())
		return
	}
	fmt.Printf("[req] time=%d id=%s\n", time.Now().UTC().UnixMilli(), req.ID)

	result, err := usercode(req.Params)
	if err != nil {
		send_err(w, req.ID, 1, err.Error())
		return
	}

	send_ok(w, req.ID, result)
}

// usercode logic
func usercode(data map[string]any) (map[string]any, error) {
	commandName, ok := data["command"].(string)
	if !ok || commandName == "" {
		return nil, fmt.Errorf("command parameter is required")
	}

	uploadURL, _ := data["upload_url"].(string)

	accessToken, _ := data["access_token"].(string)


	argsInterface, exists := data["args"]
	if !exists {
		return nil, fmt.Errorf("args parameter is required")
	}

	var args []string
	if argsList, ok := argsInterface.([]interface{}); ok {
		for _, arg := range argsList {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	} else {
		return nil, fmt.Errorf("args must be an array")
	}

	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("ai-pipe-%d", time.Now().UnixNano()))
	err := os.MkdirAll(tmpDir, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filesInterface, hasFiles := data["files"]
	if hasFiles {
		filesList, ok := filesInterface.([]interface{})
		if !ok {
			return nil, fmt.Errorf("files must be an array")
		}

		err = downloadFiles(filesList, tmpDir)
		if err != nil {
			return nil, fmt.Errorf("failed to download files: %v", err)
		}
	}

	cmd := exec.Command(commandName, args...)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+tmpDir)

	result, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command error: %s", string(result))
	}

	files, err := collectFiles(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %v", err)
	}

	uploadedFiles, err := uploadFiles(files, uploadURL, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to upload files: %v", err)
	}

	data["result"] = string(result)
	data["status"] = "success"
	data["uploaded_files"] = uploadedFiles
	return data, nil
}

func collectFiles(tmpDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func uploadFiles(files []string, uploadURL, accessToken string) ([]map[string]string, error) {
	if uploadURL == "" {
		return nil, nil
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access_token parameter is required")
	}

	var uploadedFiles []map[string]string

	for _, filePath := range files {
		err := func() error {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return fmt.Errorf("failed to stat file %s: %v", filePath, err)
			}

			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %v", filePath, err)
			}
			defer file.Close()

			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			part, err := writer.CreateFormFile("file", filepath.Base(filePath))
			if err != nil {
				return fmt.Errorf("failed to create form file: %v", err)
			}

			_, err = io.Copy(part, file)
			if err != nil {
				return fmt.Errorf("failed to copy file content: %v", err)
			}

			err = writer.Close()
			if err != nil {
				return fmt.Errorf("failed to close multipart writer: %v", err)
			}

			req, err := http.NewRequest("POST", uploadURL, &buf)
			if err != nil {
				return fmt.Errorf("failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Authorization", "Bearer "+accessToken)

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to upload file %s: %v", filePath, err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("upload failed for file %s: status %d, body: %s", filePath, resp.StatusCode, string(body))
			}

			var apiResponse struct {
				Data struct {
					ID       int    `json:"id"`
					Title    string `json:"title"`
					FileName string `json:"fileName"`
					Size     int    `json:"size"`
				} `json:"data"`
			}

			if err := json.Unmarshal(body, &apiResponse); err != nil {
				fmt.Printf("Upload OK for file %s but failed to parse response: %v\n", filePath, err)
				fmt.Printf("Response: %s\n", string(body))
				uploadedFiles = append(uploadedFiles, map[string]string{
					"filename": filepath.Base(filePath),
					"size":     fmt.Sprintf("%d", fileInfo.Size()),
					"status":   "uploaded",
				})
			} else {
				fmt.Printf("Upload OK for file %s\n", filePath)
				uploadedFiles = append(uploadedFiles, map[string]string{
					"file_name": apiResponse.Data.Title,
					"size":      fmt.Sprintf("%d", fileInfo.Size()),
					"file_link": "https://mw-api.simulator.company/v/1.0/download/" + apiResponse.Data.FileName,
				})
			}
			return nil
		}()

		if err != nil {
			uploadedFiles = append(uploadedFiles, map[string]string{
				"filename": filepath.Base(filePath),
				"error":    err.Error(),
				"status":   "failed",
			})
		}
	}

	return uploadedFiles, nil
}

func send_ok(w http.ResponseWriter, id string, result map[string]any) {
	fmt.Printf("[res] time=%d id=%s\n", time.Now().UTC().UnixMilli(), id)
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(response{
		Jsonrpc: "2.0",
		ID:      id,
		Result:  result,
	})
	if err == nil {
		// nolint
		w.Write(b)
	}
}

func send_err(w http.ResponseWriter, id string, code int, message string) {
	fmt.Printf("[res] time=%d id=%s error=%s\n", time.Now().UTC().UnixMilli(), id, message)
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(response{
		Jsonrpc: "2.0",
		ID:      id,
		Error: &respError{
			Code:    code,
			Message: message,
		},
	})
	if err == nil {
		// nolint
		w.Write(b)
	}
}

func downloadFiles(filesList []interface{}, tmpDir string) error {
	for _, fileInterface := range filesList {
		fileObj, ok := fileInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("each file must be an object")
		}

		fileLink, ok := fileObj["file_link"].(string)
		if !ok || fileLink == "" {
			return fmt.Errorf("file_link is required for each file")
		}

		fileName, ok := fileObj["file_name"].(string)
		if !ok || fileName == "" {
			return fmt.Errorf("file_name is required for each file")
		}

		resp, err := http.Get(fileLink)
		if err != nil {
			return fmt.Errorf("failed to download file %s: %v", fileName, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download file %s: HTTP %d", fileName, resp.StatusCode)
		}

		filePath := filepath.Join(tmpDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filePath, err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save file %s: %v", fileName, err)
		}

		fmt.Printf("Downloaded file: %s\n", fileName)
	}
	return nil
}

func check_panic(w http.ResponseWriter, req *http.Request) {
	if r := recover(); r != nil {
		slog.Error("logPanic", "r", r)
		send_err(w, req.Header.Get("x-request-id"), 1, fmt.Sprintf("%v", r))
	}
}
