package files

import (
	"errors"
	"io/fs"
	"testing"
	"time"
)

type fileInfo struct{ mode fs.FileMode }

func (f fileInfo) Name() string       { return "part.m3d" }
func (f fileInfo) Size() int64        { return 1 }
func (f fileInfo) Mode() fs.FileMode  { return f.mode }
func (f fileInfo) ModTime() time.Time { return time.Time{} }
func (f fileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fileInfo) Sys() any           { return nil }

func TestFileStates(t *testing.T) {
	for _, tc := range []struct {
		name string
		info fs.FileInfo
		err  error
		want string
	}{
		{"found", fileInfo{}, nil, "Найден"},
		{"missing", nil, &fs.PathError{Op: "stat", Path: `D:\\part.m3d`, Err: fs.ErrNotExist}, "Отсутствует"},
		{"denied", nil, &fs.PathError{Op: "stat", Path: `D:\\part.m3d`, Err: fs.ErrPermission}, "Нет доступа"},
		{"other", nil, errors.New("device unavailable"), "Проверить не удалось"},
		{"directory", fileInfo{mode: fs.ModeDir}, nil, "Проверить не удалось"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := check(`D:\\part.m3d`, func(string) (fs.FileInfo, error) { return tc.info, tc.err })
			if r.Status != tc.want {
				t.Fatalf("status = %q, want %q", r.Status, tc.want)
			}
		})
	}
}

func TestUnresolvedAddressesDoNotTouchFilesystem(t *testing.T) {
	for _, path := range []string{"", "part.m3d", `..\part.m3d`, `D:part.m3d`, `\part.m3d`, `\\server\part.m3d`, "https://example.test/model", "file:///D:/part.m3d", "mailto:engineer@example.test"} {
		t.Run(path, func(t *testing.T) {
			r := check(path, func(string) (fs.FileInfo, error) { t.Fatal("unexpected filesystem access"); return nil, nil })
			if r.Status == "Найден" || r.Status == "Отсутствует" {
				t.Fatalf("unexpected status %q", r.Status)
			}
		})
	}
}
