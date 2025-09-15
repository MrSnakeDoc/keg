package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"

	"gopkg.in/yaml.v3"
)

func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if info.IsDir() {
		return false, fmt.Errorf("expected a file, got a directory: %s", path)
	}
	return true, nil
}

var LookForFileInPath = DefaultLookForFileInPath

func DefaultLookForFileInPath(file string) (string, error) {
	absPath, err := exec.LookPath(file)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", file, err)
	}
	return absPath, nil
}

const (
	FileTypeJSON   = "json"
	FileTypeYAML   = "yaml"
	FileTypeBinary = "binary"
)

func FileReader(path string, fileType string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}
	if len(data) == 0 {
		return fmt.Errorf("file %s is empty", path)
	}

	switch fileType {
	case "json":
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("failed to unmarshal JSON from %s: %w", path, err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, out); err != nil {
			return fmt.Errorf("failed to unmarshal YAML from %s: %w", path, err)
		}
	default:
		return fmt.Errorf("unsupported file type %s for file %s", fileType, path)
	}
	return nil
}

func CreateFile(path string, content any, fileType string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create parent directories for %s: %w", path, err)
	}

	var data []byte
	var err error

	switch fileType {
	case FileTypeJSON:
		data, err = json.MarshalIndent(content, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for %s: %w", path, err)
		}
	case FileTypeYAML:
		data, err = yaml.Marshal(content)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML for %s: %w", path, err)
		}
	case FileTypeBinary:
		bytesContent, ok := content.([]byte)
		if !ok {
			return fmt.Errorf("invalid content type for binary file %s", path)
		}
		data = bytesContent
	default:
		return fmt.Errorf("unsupported file type %s for file %s", fileType, path)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Debug("failed to get user home directory: %w", err)
		return ""
	}
	return home
}

func WarnBrewInstallation(path string) {
	logger.Warn("⚠️ Found keg binary in %s, this may be a system installation. Proceed with caution.", path)
	if strings.Contains(path, ".linuxbrew") {
		logger.Warn("⚠️ If you want to update keg using linuxbrew, please use the command `brew update keg`.")
		logger.Warn("⚠️ If you want to install directly keg, use the command `brew uninstall keg`")
	} else {
		logger.Warn("⚠️ If you want to use the update command, please delete the keg binary in %s.", path)
	}
	logger.Warn("Then use the install script: `curl -L https://raw.githubusercontent.com/MrSnakeDoc/keg/main/scripts/install.sh | sh -`")
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func MakeFilePath(dir string, filename string) string {
	homedir := GetHomeDir()

	return filepath.Join(homedir, dir, filename)
}

type bytesRSC struct {
	data []byte
	off  int64
}

func NewBytesReadSeekCloser(b []byte) io.ReadSeekCloser {
	return &bytesRSC{data: b}
}

func (b *bytesRSC) Read(p []byte) (int, error) {
	if b.off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.off:])
	b.off += int64(n)
	return n, nil
}

func (b *bytesRSC) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case io.SeekStart:
		base = 0
	case io.SeekCurrent:
		base = b.off
	case io.SeekEnd:
		base = int64(len(b.data))
	default:
		return 0, syscall.EINVAL
	}
	npos := base + offset
	if npos < 0 {
		return 0, syscall.EINVAL
	}
	b.off = npos
	return b.off, nil
}

func (b *bytesRSC) Close() error { return nil }

func WriteFileAtomic(tmpPath, finalPath string, r io.Reader) error {
	// Create tmp
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(tmp, r)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()

	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if syncErr != nil {
		_ = os.Remove(tmpPath)
		return syncErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}

	// Rename atomically
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	// fsync directory for durability
	return fsyncDir(filepath.Dir(finalPath))
}

func WriteJSONAtomic(path string, v any) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "")
	if err := enc.Encode(v); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func fsyncDir(dir string) error {
	df, err := os.Open(dir)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := df.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	// On non-Unix, Sync may be no-op; fine.
	if f, ok := any(df).(interface{ Sync() error }); ok {
		_ = f.Sync()
	}
	return err
}
