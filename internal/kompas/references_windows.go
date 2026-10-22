package kompas

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

var automation = windows.NewLazySystemDLL("oleaut32.dll")
var arrayDimensions = automation.NewProc("SafeArrayGetDim")
var arrayLower = automation.NewProc("SafeArrayGetLBound")
var arrayUpper = automation.NewProc("SafeArrayGetUBound")
var arrayElement = automation.NewProc("SafeArrayGetElement")
var freeBSTR = automation.NewProc("SysFreeString")

func readReferences(doc *ole.IDispatch) ([]string, error) {
	interfaceID := ole.NewGUID("{58890FE8-E671-4561-994A-600DD29032E4}")
	links, err := doc.QueryInterface(interfaceID)
	if err != nil {
		return nil, fmt.Errorf("интерфейс IKompasDocument1 недоступен: %w", err)
	}
	if links == nil {
		return nil, errors.New("пустой интерфейс IKompasDocument1")
	}
	defer links.Release()
	value, err := oleutil.GetProperty(links, "ExternalFilesNames", true)
	var dispatchErr *ole.OleError
	if errors.As(err, &dispatchErr) && uint32(dispatchErr.Code()) == 0x8002000e {
		if value != nil {
			value.Clear()
		}
		value, err = oleutil.GetProperty(links, "ExternalFilesNames")
	}
	if value != nil {
		defer value.Clear()
	}
	if err != nil {
		return nil, fmt.Errorf("ExternalFilesNames: чтение не поддержано или завершилось ошибкой: %w", err)
	}
	if value == nil {
		return nil, errors.New("ExternalFilesNames не вернул VARIANT")
	}
	if value.VT == ole.VT_EMPTY || value.VT == ole.VT_NULL {
		return nil, errors.New("ExternalFilesNames вернул пустое значение вместо массива; отсутствие ссылок не подтверждено")
	}
	return readStringArray(value)
}

func arrayResult(result uintptr) error {
	if int32(result) < 0 {
		return ole.NewError(uintptr(uint32(result)))
	}
	return nil
}

func readStringArray(value *ole.VARIANT) ([]string, error) {
	if value.VT != ole.VT_ARRAY|ole.VT_BSTR && value.VT != ole.VT_ARRAY|ole.VT_VARIANT {
		return nil, fmt.Errorf("ExternalFilesNames: неожиданный тип VARIANT %d", value.VT)
	}
	array := value.ToArray()
	if array == nil || array.Array == nil {
		return nil, errors.New("пустой указатель SAFEARRAY")
	}
	defer runtime.KeepAlive(value)
	ptr := uintptr(unsafe.Pointer(array.Array))
	for _, proc := range []*windows.LazyProc{arrayDimensions, arrayLower, arrayUpper, arrayElement, freeBSTR} {
		if err := proc.Find(); err != nil {
			return nil, err
		}
	}
	dimensions, _, _ := arrayDimensions.Call(ptr)
	if dimensions != 1 {
		return nil, fmt.Errorf("ожидался одномерный SAFEARRAY, получено %d", dimensions)
	}
	var lower, upper int32
	result, _, _ := arrayLower.Call(ptr, 1, uintptr(unsafe.Pointer(&lower)))
	if err := arrayResult(result); err != nil {
		return nil, err
	}
	result, _, _ = arrayUpper.Call(ptr, 1, uintptr(unsafe.Pointer(&upper)))
	if err := arrayResult(result); err != nil {
		return nil, err
	}
	count := int64(upper) - int64(lower) + 1
	if count < 0 || count > 100000 {
		return nil, fmt.Errorf("недопустимый размер списка ссылок: %d", count)
	}
	paths := make([]string, 0, int(count))
	for offset := int64(0); offset < count; offset++ {
		index := int32(int64(lower) + offset)
		var text string
		if value.VT == ole.VT_ARRAY|ole.VT_BSTR {
			var bstr *int16
			result, _, _ = arrayElement.Call(ptr, uintptr(unsafe.Pointer(&index)), uintptr(unsafe.Pointer(&bstr)))
			if err := arrayResult(result); err != nil {
				return nil, err
			}
			text = ole.BstrToString((*uint16)(unsafe.Pointer(bstr)))
			freeBSTR.Call(uintptr(unsafe.Pointer(bstr)))
		} else {
			var item ole.VARIANT
			result, _, _ = arrayElement.Call(ptr, uintptr(unsafe.Pointer(&index)), uintptr(unsafe.Pointer(&item)))
			if err := arrayResult(result); err != nil {
				return nil, err
			}
			if item.VT != ole.VT_BSTR {
				item.Clear()
				return nil, fmt.Errorf("ссылка %d не является строкой", index)
			}
			text = item.ToString()
			if err := item.Clear(); err != nil {
				return nil, err
			}
		}
		paths = append(paths, text)
	}
	return paths, nil
}
