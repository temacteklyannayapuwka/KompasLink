package files

import (
	"errors"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"kompaslink/internal/diagnostic"
)

var scheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

func Check(path string) diagnostic.Reference {
	return check(path, os.Stat)
}

func check(path string, stat func(string) (fs.FileInfo, error)) diagnostic.Reference {
	r := diagnostic.Reference{Path: path, Type: "Не определён API", Status: "Проверить не удалось"}
	if strings.TrimSpace(path) == "" {
		r.Detail = "API вернул пустой адрес"
		return r
	}
	drive := len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':'
	if !drive && scheme.MatchString(path) {
		r.Status = "Не проверяется"
		r.Detail = "Адрес с URI-схемой; не проверяется как обычный файл"
		return r
	}
	if strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//") {
		r.Detail = "Сетевой или специальный путь; локальный режим его не проверяет"
		return r
	}
	if !drive || len(path) < 3 || (path[2] != '\\' && path[2] != '/') {
		r.Detail = "Относительный путь; правила его разрешения в КОМПАС-18 не подтверждены"
		return r
	}
	info, err := stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		r.Status = "Отсутствует"
	case errors.Is(err, fs.ErrPermission):
		r.Status = "Нет доступа"
	case err != nil:
		r.Detail = err.Error()
	case info == nil:
		r.Detail = "Файловая система не вернула сведения"
	case !info.Mode().IsRegular():
		r.Detail = "Адрес указывает не на обычный файл"
	default:
		r.Status = "Найден"
	}
	return r
}
