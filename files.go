package pufferpanel

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const MaxRecursivePath = 256

func JoinPath(paths ...string) string {
	return filepath.Join(paths...)
}

func EnsureAccess(source string, prefix string) bool {
	//first resolve the correct source path so we can compare with the wanted path
	expected, err := findFullPath(prefix)
	if err != nil && !os.IsNotExist(err) {
		return false
	}

	expected, err = filepath.Abs(expected)
	if err != nil && !os.IsNotExist(err) {
		return false
	}

	replacement, err := findFullPath(source)
	if err != nil && !os.IsNotExist(err) {
		return false
	}
	replacement, err = filepath.Abs(replacement)
	if err != nil {
		return false
	}

	return strings.HasPrefix(replacement, expected)
}

func RemoveInvalidSymlinks(files []os.FileInfo, sourceFolder, prefix string) []os.FileInfo {
	i := 0
	for _, v := range files {
		if v.Mode()&os.ModeSymlink != 0 {
			if !EnsureAccess(sourceFolder+string(os.PathSeparator)+v.Name(), prefix) {
				continue
			}
		}
		files[i] = v
		i++
	}

	return files[:i]
}

func CopyFile(src, dest string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer Close(source)

	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return err
	}
	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer Close(destination)
	_, err = io.Copy(destination, source)
	return err
}

func findFullPath(source string) (string, error) {
	fullPath, err := filepath.EvalSymlinks(source)

	if err == nil {
		return fullPath, err
	}

	//if file doesn't exist, then filepath doesn't resolve properly, so check backwards
	if os.IsNotExist(err) {
		var updatePath string
		dir, filename := filepath.Split(source)
		suffix := string(os.PathSeparator) + filename

		i := 0
		for i < MaxRecursivePath && dir != "" {
			dirFullPath, err := filepath.EvalSymlinks(dir)
			if err != nil && os.IsNotExist(err) {
				//update our mapping to look farther down
				suffix = filepath.Base(dir) + string(os.PathSeparator) + suffix
				dir = filepath.Dir(dir)
			} else if err != nil {
				return "", err
			} else {
				//we found a good path!
				updatePath = dirFullPath + string(os.PathSeparator) + suffix
				break
			}
			i++

			if i == MaxRecursivePath {
				return "", errors.New("path too recursive")
			}
		}

		updatePath, err = filepath.Abs(updatePath)
		return updatePath, err

	}

	return "", err
}
