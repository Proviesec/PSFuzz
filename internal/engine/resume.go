package engine

import (
	"encoding/gob"
	"encoding/json"
	"os"
	"strings"
	"time"
)

func loadResume(path string) map[string]struct{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]struct{}{}
	}
	if strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		type resumeFile struct {
			URLs []string `json:"urls"`
		}
		var rf resumeFile
		if err := json.Unmarshal(data, &rf); err == nil {
			out := map[string]struct{}{}
			for _, u := range rf.URLs {
				u = strings.TrimSpace(u)
				if u != "" {
					out[u] = struct{}{}
				}
			}
			return out
		}
	}
	if strings.HasSuffix(path, ".bin") {
		if urls, err := readResumeBin(path); err == nil {
			out := map[string]struct{}{}
			for _, u := range urls {
				u = strings.TrimSpace(u)
				if u != "" {
					out[u] = struct{}{}
				}
			}
			return out
		}
	}
	lines := strings.Split(string(data), "\n")
	out := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out[line] = struct{}{}
		}
	}
	return out
}

func readResumeBin(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var urls []string
	if err := dec.Decode(&urls); err != nil {
		return nil, err
	}
	return urls, nil
}

func writeResume(path string, visited map[string]struct{}) error {
	type resumeFile struct {
		Time string   `json:"time"`
		URLs []string `json:"urls"`
	}
	urls := make([]string, 0, len(visited))
	for url := range visited {
		urls = append(urls, url)
	}
	data, err := json.MarshalIndent(resumeFile{Time: time.Now().Format(time.RFC3339), URLs: urls}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	if err := writeResumeBin(path+".bin", urls); err != nil {
		return err
	}
	return nil
}

func writeResumeBin(path string, urls []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return enc.Encode(urls)
}
