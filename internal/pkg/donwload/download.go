package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func Download(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// 根据URL确定文件扩展名
	ext := ".tmp"
	if strings.HasSuffix(url, ".zip") {
		ext = ".zip"
	} else if strings.HasSuffix(url, ".tar.gz") {
		ext = ".tar.gz"
	} else if strings.HasSuffix(url, ".tgz") {
		ext = ".tgz"
	}

	tempFile, err := os.CreateTemp("", "package-*"+ext)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}
