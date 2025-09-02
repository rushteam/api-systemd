package artifact

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractZipFile 解压ZIP文件并返回顶级文件夹名
func extractZipFile(src, dest string) ([]string, error) {
	var topLevelFolders []string
	folderSet := make(map[string]struct{})

	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path: %s", fPath)
		}

		// 获取顶级目录/文件夹
		relativePath, err := filepath.Rel(dest, fPath)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(relativePath, string(os.PathSeparator))
		if len(parts) > 0 && parts[0] != "." && parts[0] != ".." {
			if _, exists := folderSet[parts[0]]; !exists {
				folderSet[parts[0]] = struct{}{}
				topLevelFolders = append(topLevelFolders, parts[0])
			}
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, f.FileInfo().Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return nil, err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return nil, err
		}
	}

	return topLevelFolders, nil
}

// extractTarFile 解压TAR文件并返回顶级文件夹名
func extractTarFile(src, dest string) ([]string, error) {
	var topLevelFolders []string
	folderSet := make(map[string]struct{})

	file, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var reader io.Reader = file

	// 如果是gzip压缩的tar文件
	if strings.HasSuffix(src, ".gz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path: %s", target)
		}

		// 获取顶级目录/文件夹
		relativePath, err := filepath.Rel(dest, target)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(relativePath, string(os.PathSeparator))
		if len(parts) > 0 && parts[0] != "." && parts[0] != ".." {
			if _, exists := folderSet[parts[0]]; !exists {
				folderSet[parts[0]] = struct{}{}
				topLevelFolders = append(topLevelFolders, parts[0])
			}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil, err
			}

			outFile, err := os.Create(target)
			if err != nil {
				return nil, err
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return nil, err
			}

			outFile.Close()

			// 设置文件权限
			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				return nil, err
			}
		}
	}

	return topLevelFolders, nil
}

