package engine

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type rawRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

func parseRawRequest(path string) (*rawRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(data), "\r\n\r\n", 2)
	if len(parts) == 1 {
		parts = strings.SplitN(string(data), "\n\n", 2)
	}
	headerBlock := parts[0]
	body := ""
	if len(parts) == 2 {
		body = parts[1]
	}
	lines := strings.Split(headerBlock, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("invalid request file")
	}
	requestLine := strings.TrimSpace(lines[0])
	fields := strings.Fields(requestLine)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid request line")
	}
	headers := map[string]string{}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return &rawRequest{
		Method:  fields[0],
		Path:    fields[1],
		Headers: headers,
		Body:    body,
	}, nil
}

func joinBaseAndPathWithProto(base, path string, proto string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	u, err := url.Parse(base)
	if err != nil {
		return base + path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	scheme := u.Scheme
	if proto != "" {
		scheme = proto
	}
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + u.Host + path
}

func mergeHeaders(base map[string]string, extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return base
	}
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
