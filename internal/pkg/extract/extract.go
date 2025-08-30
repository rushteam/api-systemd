package extract

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

func Extract(src, dest string) ([]string, error) {
	var topLevelFolders []string
	var err error
	if strings.HasSuffix(src, ".zip") {
		topLevelFolders, err = unzip(src, dest)
	} else if strings.HasSuffix(src, ".tgz") || strings.HasSuffix(src, ".tar.gz") {
		topLevelFolders, err = untar(src, dest)
	} else {
		return nil, fmt.Errorf("unsupported file type: %s", src)
	}
	if err != nil {
		return nil, err
	}
	return topLevelFolders, nil
}

// unzip extracts files from a .zip archive and returns the top-level folder name(s)
func unzip(src, dest string) ([]string, error) {
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

		// Capture the top-level directory/folder
		relativePath, err := filepath.Rel(dest, fPath)
		if err != nil {
			return nil, err
		}
		topFolder := strings.SplitN(relativePath, string(os.PathSeparator), 2)[0]
		if _, found := folderSet[topFolder]; !found {
			folderSet[topFolder] = struct{}{}
			topLevelFolders = append(topLevelFolders, topFolder)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return nil, err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
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

// untar extracts files from a .tgz or .tar.gz archive and returns the top-level folder name(s)
func untar(src, dest string) ([]string, error) {
	var topLevelFolders []string
	folderSet := make(map[string]struct{})

	file, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		fPath := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(fPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path: %s", fPath)
		}

		// Capture the top-level directory/folder
		relativePath, err := filepath.Rel(dest, fPath)
		if err != nil {
			return nil, err
		}
		topFolder := strings.SplitN(relativePath, string(os.PathSeparator), 2)[0]
		if _, found := folderSet[topFolder]; !found {
			folderSet[topFolder] = struct{}{}
			topLevelFolders = append(topLevelFolders, topFolder)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return nil, err
			}
			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return nil, err
			}
			outFile.Close()
		default:
			return nil, fmt.Errorf("unsupported type: %v in %s", header.Typeflag, header.Name)
		}
	}

	return topLevelFolders, nil
}
