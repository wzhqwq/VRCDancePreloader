package config_widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type DownloadableBinaryGui struct {
	interactive_widgets.LifeCycleWidget

	downloadable *local_executables.DownloadableBinary

	setting      interactive.StatefulSetting[string]
	inputSetting interactive.StatefulSetting[string]
	checkSetting interactive.StatefulSetting[bool]

	progressChanged bool
	stateChanged    bool
	versionChanged  bool
}

func NewDownloadableBinaryGui(name string, setting interactive.StatefulSetting[string]) *DownloadableBinaryGui {
	d := &DownloadableBinaryGui{
		downloadable: local_executables.Get(name),
		setting:      setting,
		inputSetting: interactive.NewDerivedSetting(
			setting,
			func(mixed string) string {
				if mixed == "<vrcdp>" {
					return ""
				}
				return mixed
			},
			func(p string) string {
				return p
			},
		),
		checkSetting: interactive.NewDerivedSetting(
			setting,
			func(mixed string) bool {
				return mixed != "<vrcdp>"
			},
			func(custom bool) string {
				if custom {
					return ""
				}
				return "<vrcdp>"
			},
		),
	}
	d.AddLifeCycleFn(d.loop)
	d.ExtendBaseWidget(d)

	return d
}

func (d *DownloadableBinaryGui) UpgradeText() string {
	if d.downloadable.State != local_executables.BinUpdateAvailable {
		return i18n.T("btn_check_update")
	}

	if d.downloadable.Info.Exists {
		return i18n.T("btn_upgrade")
	}
	return i18n.T("btn_download")
}

func (d *DownloadableBinaryGui) SizeText() string {
	if d.downloadable.State == local_executables.BinCheckingLocal {
		return i18n.T("placeholder_checking_local")
	}
	if d.downloadable.State == local_executables.BinDownloaded {
		return i18n.T("placeholder_waiting_unlock")
	}
	if d.downloadable.Info.Exists {
		return utils.PrettyByteSize(d.downloadable.Info.Size)
	}
	return i18n.T("placeholder_not_downloaded")
}

func (d *DownloadableBinaryGui) CreateRenderer() fyne.WidgetRenderer {
	name := canvas.NewText(d.downloadable.Name, theme.Color(theme.ColorNameForeground))
	name.TextStyle.Bold = true
	name.TextSize = 16

	size := canvas.NewText(d.SizeText(), theme.Color(theme.ColorNamePlaceHolder))
	size.TextSize = 12

	errorText := canvas.NewText("", theme.Color(theme.ColorNameError))
	errorText.TextSize = 12

	localVersion := canvas.NewText(d.downloadable.Info.Version, theme.Color(theme.ColorNamePlaceHolder))
	localVersion.TextSize = 12

	upgradeVersion := canvas.NewText("", theme.Color(theme.ColorNamePlaceHolder))
	upgradeVersion.TextSize = 12

	removeButton := widget.NewButton(i18n.T("btn_remove"), func() {
		go d.downloadable.Remove()
	})
	removeButton.Importance = widget.DangerImportance

	upgradeButton := widget.NewButton(d.UpgradeText(), func() {
		if d.downloadable.State == local_executables.BinUpdateAvailable {
			go d.downloadable.Upgrade()
		} else {
			go d.downloadable.CheckUpdates()
		}
	})
	upgradeButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton(i18n.T("btn_cancel_download"), func() {
		d.downloadable.CancelDownload()
	})

	bar := widgets.NewDownloadBar()

	versionLine := container.NewHBox(localVersion, upgradeVersion)
	operations := container.NewBorder(
		nil, container.NewVBox(bar, errorText),
		container.NewVBox(size, versionLine),
		container.NewHBox(removeButton, upgradeButton, cancelButton),
	)

	customInput := input.NewInputWithSave(d.inputSetting, i18n.T("label_executable_path"))
	customCb := input.NewCheck(i18n.T("label_executable_use_custom"), d.checkSetting)

	box := container.NewVBox(
		container.NewHBox(name, customCb),
		interactive_widgets.NewAvailableWhen(customInput, d.checkSetting),
		interactive_widgets.NewAvailableWhenNot(operations, d.checkSetting),
	)

	r := &downloadableBinaryRenderer{
		d: d,

		nameText:           name,
		sizeText:           size,
		errorText:          errorText,
		localVersionText:   localVersion,
		upgradeVersionText: upgradeVersion,

		customPathInput: customInput,

		removeButton:  removeButton,
		upgradeButton: upgradeButton,
		cancelButton:  cancelButton,

		bar: bar,

		box:        box,
		operations: operations,
	}

	r.updateState()
	r.updateProgress()
	r.updateVersion()
	r.Created(d)

	return r
}

func (d *DownloadableBinaryGui) loop(stopCh <-chan struct{}) {
	ch := d.downloadable.Subscribe()
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case t := <-ch.Channel:
			switch t {
			case local_executables.BinProgress:
				d.progressChanged = true
			case local_executables.BinState:
				d.stateChanged = true
			case local_executables.BinVersion:
				d.versionChanged = true
			}
			fyne.Do(func() {
				d.Refresh()
			})
		}
	}
}

type downloadableBinaryRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

	d *DownloadableBinaryGui

	nameText           *canvas.Text
	sizeText           *canvas.Text
	errorText          *canvas.Text
	localVersionText   *canvas.Text
	upgradeVersionText *canvas.Text

	customPathInput *input.InputWithSave

	removeButton  *widget.Button
	upgradeButton *widget.Button
	cancelButton  *widget.Button

	bar *widgets.DownloadBar

	box        *fyne.Container
	operations *fyne.Container
}

func (r *downloadableBinaryRenderer) MinSize() fyne.Size {
	return r.box.MinSize().AddWidthHeight(theme.Padding()*2, 0)
}

func (r *downloadableBinaryRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	r.box.Move(fyne.NewPos(p, 0))
	r.box.Resize(size.SubtractWidthHeight(p*2, 0))
}

func (r *downloadableBinaryRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.box}
}

func (r *downloadableBinaryRenderer) Refresh() {
	if r.d.stateChanged {
		r.d.stateChanged = false
		r.updateState()
	}
	if r.d.progressChanged {
		r.d.progressChanged = false
		r.updateProgress()
	}
	if r.d.versionChanged {
		r.d.versionChanged = false
		r.updateVersion()
	}
	r.box.Refresh()
}

func (r *downloadableBinaryRenderer) updateState() {
	state := r.d.downloadable.State

	if state != local_executables.BinCheckingLocal && r.d.downloadable.Info.Exists {
		r.removeButton.Show()
		r.localVersionText.Show()
	} else {
		r.removeButton.Hide()
		r.localVersionText.Hide()
	}

	if r.d.downloadable.HasUpdates() {
		r.upgradeVersionText.Text = r.d.downloadable.UpdateText()
		r.upgradeVersionText.Show()
	} else {
		r.upgradeVersionText.Hide()
	}

	r.upgradeButton.Text = r.d.UpgradeText()
	r.sizeText.Text = r.d.SizeText()

	switch state {
	case local_executables.BinCheckingLocal:
		r.upgradeButton.Hide()
		r.cancelButton.Hide()
	case local_executables.BinInitial, local_executables.BinCheckingUpdates, local_executables.BinUpdateAvailable:
		r.upgradeButton.Show()
		r.cancelButton.Hide()
	case local_executables.BinDownloading:
		r.upgradeButton.Hide()
		r.cancelButton.Show()
	}

	if state == local_executables.BinCheckingUpdates {
		r.upgradeButton.Disable()
	} else {
		r.upgradeButton.Enable()
	}

	if state == local_executables.BinDownloading {
		r.bar.Show()
	} else {
		r.bar.Hide()
	}

	if r.d.downloadable.Error != nil {
		r.errorText.Text = r.d.downloadable.Error.Error()
		r.errorText.Show()
	} else {
		r.errorText.Hide()
	}
}

func (r *downloadableBinaryRenderer) updateProgress() {
	task := r.d.downloadable.Task
	if task == nil {
		return
	}
	r.bar.SetProgress(task.TotalSize, task.DownloadedSize, task.Speed(), task.RemainTime())
}

func (r *downloadableBinaryRenderer) updateVersion() {
	r.localVersionText.Text = r.d.downloadable.Info.Version
}
