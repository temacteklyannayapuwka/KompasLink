package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"
	d "github.com/lxn/walk/declarative"

	"kompaslink/internal/diagnostic"
	"kompaslink/internal/kompas"
)

type result struct {
	report diagnostic.Report
	err    error
}

type window struct {
	main       *walk.MainWindow
	connection *walk.LineEdit
	version    *walk.LineEdit
	path       *walk.TextEdit
	state      *walk.LineEdit
	properties *walk.LineEdit
	refresh    *walk.PushButton
	results    *walk.TextEdit
	busy       bool
}

func Run() error {
	w := &window{}
	err := (d.MainWindow{
		AssignTo: &w.main,
		Title:    "KompasLink — диагностика КОМПАС-3D",
		MinSize:  d.Size{Width: 700, Height: 570},
		Size:     d.Size{Width: 820, Height: 650},
		Layout:   d.VBox{Margins: d.Margins{Left: 18, Top: 18, Right: 18, Bottom: 18}, Spacing: 12},
		Children: []d.Widget{
			d.Label{Text: "KompasLink", Font: d.Font{PointSize: 18, Bold: true}},
			d.Label{Text: "Диагностика активного документа • Только чтение"},
			d.GroupBox{
				Title:  "Подключение",
				Layout: d.Grid{Columns: 2, Spacing: 8},
				Children: []d.Widget{
					d.Label{Text: "Состояние:"},
					d.LineEdit{AssignTo: &w.connection, ReadOnly: true, Text: "Ещё не проверено"},
					d.Label{Text: "Версия КОМПАС:"},
					d.LineEdit{AssignTo: &w.version, ReadOnly: true, Text: "—"},
				},
			},
			d.GroupBox{
				Title:  "Активный документ",
				Layout: d.Grid{Columns: 2, Spacing: 8},
				Children: []d.Widget{
					d.Label{Text: "Путь:"},
					d.TextEdit{AssignTo: &w.path, ReadOnly: true, Text: "—", MinSize: d.Size{Height: 56}},
					d.Label{Text: "Состояние:"},
					d.LineEdit{AssignTo: &w.state, ReadOnly: true, Text: "—"},
					d.Label{Text: "Свойства:"},
					d.LineEdit{AssignTo: &w.properties, ReadOnly: true, Text: "—"},
				},
			},
			d.Composite{
				Layout: d.HBox{},
				Children: []d.Widget{
					d.PushButton{AssignTo: &w.refresh, Text: "Обновить диагностику", OnClicked: w.start},
					d.HSpacer{},
				},
			},
			d.Label{Text: "Результаты текущего запуска:"},
			d.TextEdit{
				AssignTo: &w.results, ReadOnly: true, VScroll: true,
				MinSize: d.Size{Height: 130}, StretchFactor: 1,
				Text: "Откройте тестовую копию .a3d или .m3d в КОМПАС и нажмите «Обновить диагностику».\r\nСовместимость с КОМПАС-18 на реальной установке ещё не подтверждена. Внешние ссылки пока не читаются.",
			},
		},
	}).Create()
	if err != nil {
		return err
	}
	defer w.main.Dispose()

	w.main.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if w.busy {
			*canceled = true
			w.append("Дождитесь завершения диагностики перед закрытием окна.")
		}
	})
	w.main.Run()
	return nil
}

func (w *window) start() {
	if w.busy {
		return
	}
	w.busy = true
	w.refresh.SetEnabled(false)
	w.connection.SetText("Проверка…")
	w.version.SetText("—")
	w.path.SetText("—")
	w.state.SetText("—")
	w.properties.SetText("—")
	w.append("Подключение к запущенному КОМПАС…")
	go func() {
		report, err := kompas.Probe()
		w.main.Synchronize(func() {
			w.busy = false
			w.refresh.SetEnabled(true)
			w.show(result{report: report, err: err})
		})
	}()
}

func (w *window) show(value result) {
	r := value.report
	if r.Connected {
		w.connection.SetText("Подключено к запущенному экземпляру")
	} else {
		w.connection.SetText("Подключение не установлено")
	}
	if r.Version != nil {
		v := r.Version
		version := fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Release, v.Build)
		w.version.SetText(version)
		if v.Major != 18 {
			w.append("Подключён КОМПАС версии " + version + ". Целевая версия проекта — 18.")
		}
	}
	if value.err != nil {
		w.state.SetText("Сведения не получены")
		w.append("Ошибка: " + value.err.Error())
		return
	}
	if r.Document == nil {
		w.state.SetText("Нет активного документа")
		w.append("Подключение выполнено. Откройте документ в КОМПАС и обновите диагностику.")
		return
	}
	doc := r.Document
	path := doc.Path
	if strings.TrimSpace(path) == "" {
		path = doc.Name
	}
	w.path.SetText(path)
	state := "Сохранённый путь получен"
	if !doc.HasSavedPath {
		state = "Документ без сохранённого пути"
	}
	if doc.Changed {
		state += "; есть несохранённые изменения"
	}
	w.state.SetText(state)
	mode := "Обычный режим"
	if doc.ReadOnly {
		mode = "Только для чтения"
	}
	w.properties.SetText(fmt.Sprintf("%s; тип API: %d", mode, doc.TypeID))
	w.append("Диагностика завершена: " + doc.Name + ". Файл не изменялся.")
}

func (w *window) append(message string) {
	text := w.results.Text()
	if len(text) > 24000 {
		text = ""
	}
	w.results.SetText(text + "\r\n[" + time.Now().Format("15:04:05") + "] " + message)
}
