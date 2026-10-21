package kompas

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"kompaslink/internal/diagnostic"
)

func Probe() (report diagnostic.Report, err error) {
	report.ProgID = "KOMPAS.Application.5"
	report.State = "error"
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if initErr := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); initErr != nil {
		var comErr *ole.OleError
		if !errors.As(initErr, &comErr) || comErr.Code() != 1 {
			return report, fmt.Errorf("инициализация COM STA: %w", initErr)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.GetActiveObject(report.ProgID)
	if err != nil {
		return report, fmt.Errorf("подключение к запущенному КОМПАС: %w; запустите один экземпляр с теми же правами, что и утилита", err)
	}
	if unknown == nil {
		return report, errors.New("GetActiveObject вернул пустой объект")
	}
	defer unknown.Release()

	base, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return report, fmt.Errorf("получение IDispatch API 5: %w", err)
	}
	if base == nil {
		return report, errors.New("пустой IDispatch API 5")
	}
	defer base.Release()
	report.Connected = true

	version, err := readVersion(base)
	if err != nil {
		return report, err
	}
	report.Version = &version

	appValue, err := oleutil.CallMethod(base, "ksGetApplication7")
	if appValue != nil {
		defer appValue.Clear()
	}
	if err != nil {
		return report, fmt.Errorf("ksGetApplication7: %w", err)
	}
	if appValue == nil || appValue.ToIDispatch() == nil {
		return report, errors.New("ksGetApplication7 не вернул IDispatch")
	}

	docValue, err := oleutil.GetProperty(appValue.ToIDispatch(), "ActiveDocument")
	if docValue != nil {
		defer docValue.Clear()
	}
	if err != nil {
		return report, fmt.Errorf("ActiveDocument: %w", err)
	}
	if docValue == nil {
		return report, errors.New("ActiveDocument не вернул VARIANT")
	}
	if docValue.VT == ole.VT_EMPTY || docValue.VT == ole.VT_NULL || (docValue.VT == ole.VT_DISPATCH && docValue.Val == 0) {
		report.State = "no_active_document"
		return report, nil
	}
	doc := docValue.ToIDispatch()
	if doc == nil {
		return report, fmt.Errorf("ActiveDocument: неожиданный тип VARIANT %d", docValue.VT)
	}

	info := diagnostic.Document{}
	if info.Name, err = stringProperty(doc, "Name"); err != nil {
		return report, err
	}
	if info.Directory, err = stringProperty(doc, "Path"); err != nil {
		return report, err
	}
	if info.Path, err = stringProperty(doc, "PathName"); err != nil {
		return report, err
	}
	if info.TypeID, err = intProperty(doc, "DocumentType"); err != nil {
		return report, err
	}
	if info.Changed, err = boolProperty(doc, "Changed"); err != nil {
		return report, err
	}
	if info.ReadOnly, err = boolProperty(doc, "ReadOnly"); err != nil {
		return report, err
	}
	info.HasSavedPath = strings.TrimSpace(info.Directory) != "" && filepath.IsAbs(info.Path)
	report.Document = &info
	report.State = "document_observed"
	if !info.HasSavedPath {
		report.State = "document_without_saved_path"
	} else if info.Changed {
		report.State = "document_has_unsaved_changes"
	}
	return report, nil
}

func readVersion(base *ole.IDispatch) (diagnostic.Version, error) {
	var version diagnostic.Version
	value, err := oleutil.CallMethod(base, "ksGetSystemVersion", &version.Major, &version.Minor, &version.Release, &version.Build)
	if value != nil {
		defer value.Clear()
	}
	if err != nil {
		return version, fmt.Errorf("ksGetSystemVersion: %w", err)
	}
	if value == nil || value.VT != ole.VT_I4 || int32(value.Val) != 1 || version.Major <= 0 {
		return version, errors.New("ksGetSystemVersion не подтвердил получение версии")
	}
	return version, nil
}

func stringProperty(doc *ole.IDispatch, name string) (string, error) {
	value, err := oleutil.GetProperty(doc, name)
	if value != nil {
		defer value.Clear()
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	if value == nil || value.VT != ole.VT_BSTR {
		return "", fmt.Errorf("%s: ожидалась строка BSTR", name)
	}
	return value.ToString(), nil
}

func intProperty(doc *ole.IDispatch, name string) (int32, error) {
	value, err := oleutil.GetProperty(doc, name)
	if value != nil {
		defer value.Clear()
	}
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if value == nil || value.VT != ole.VT_I4 {
		return 0, fmt.Errorf("%s: ожидалось число LONG", name)
	}
	return int32(value.Val), nil
}

func boolProperty(doc *ole.IDispatch, name string) (bool, error) {
	value, err := oleutil.GetProperty(doc, name)
	if value != nil {
		defer value.Clear()
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	if value == nil || value.VT != ole.VT_BOOL {
		return false, fmt.Errorf("%s: ожидался VARIANT_BOOL", name)
	}
	return value.Val != 0, nil
}
