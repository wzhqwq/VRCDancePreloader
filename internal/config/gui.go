package config

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"strconv"
)

func CreateSettingsContainer() fyne.CanvasObject {
	wholeContent := container.NewVBox(
		createProxySettingsContent(),
		widget.NewSeparator(),
		createKeySettingsContent(),
		widget.NewSeparator(),
		createYoutubeSettingsContent(),
		widget.NewSeparator(),
		createPreloadSettingsContent(),
		widget.NewSeparator(),
		createDownloadSettingsContent(),
		widget.NewSeparator(),
		createCacheSettingsContent(),
	)
	scroll := container.NewVScroll(wholeContent)
	scroll.SetMinSize(fyne.NewSize(400, 300))
	scroll.Refresh()
	return scroll
}

func createProxySettingsContent() fyne.CanvasObject {
	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_proxy")))
	wholeContent.Add(NewProxyInput(config.Proxy.ProxyControllers["pypydance-api"], i18n.T("label_pypy_proxy")))
	wholeContent.Add(NewProxyInput(config.Proxy.ProxyControllers["youtube-video"], i18n.T("label_yt_video_proxy")))
	wholeContent.Add(NewProxyInput(config.Proxy.ProxyControllers["youtube-api"], i18n.T("label_yt_api_proxy")))
	wholeContent.Add(NewProxyInput(config.Proxy.ProxyControllers["youtube-image"], i18n.T("label_yt_image_proxy")))

	return wholeContent
}

func createKeySettingsContent() fyne.CanvasObject {
	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_keys")))

	youtubeKeyInput := widgets.NewInputWithSave(config.Keys.Youtube, i18n.T("label_yt_api_key"))
	youtubeKeyInput.OnSave = func() {
		config.Keys.Youtube = youtubeKeyInput.Value
		SaveConfig()
	}

	wholeContent.Add(youtubeKeyInput)
	return wholeContent
}

func createYoutubeSettingsContent() fyne.CanvasObject {
	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_youtube")))

	enableApiCheck := widget.NewCheck(i18n.T("label_yt_api_enable"), func(b bool) {
		if config.Youtube.EnableApi == b {
			return
		}
		config.Youtube.EnableApi = b
		SaveConfig()
		if b && config.Proxy.ProxyControllers["youtube-api"].Status != ProxyStatusOk {
			config.Proxy.ProxyControllers["youtube-api"].Test()
		}
	})
	enableApiCheck.Checked = config.Youtube.EnableApi
	wholeContent.Add(enableApiCheck)

	enableThumbnailCheck := widget.NewCheck(i18n.T("label_yt_thumbnail_enable"), func(b bool) {
		if config.Youtube.EnableThumbnail == b {
			return
		}
		config.Youtube.EnableThumbnail = b
		SaveConfig()
		if b && config.Proxy.ProxyControllers["youtube-image"].Status != ProxyStatusOk {
			config.Proxy.ProxyControllers["youtube-image"].Test()
		}
	})
	enableThumbnailCheck.Checked = config.Youtube.EnableThumbnail
	wholeContent.Add(enableThumbnailCheck)

	enableVideoCheck := widget.NewCheck(i18n.T("label_yt_video_enable"), func(b bool) {
		if config.Youtube.EnableVideo == b {
			return
		}
		config.Youtube.EnableVideo = b
		SaveConfig()
		if b && config.Proxy.ProxyControllers["youtube-video"].Status != ProxyStatusOk {
			config.Proxy.ProxyControllers["youtube-video"].Test()
		}
	})
	enableVideoCheck.Checked = config.Youtube.EnableVideo
	wholeContent.Add(enableVideoCheck)

	return wholeContent
}

func createPreloadSettingsContent() fyne.CanvasObject {
	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_preload")))

	maxPreloadInput := widgets.NewInputWithSave(strconv.Itoa(config.Preload.MaxPreload), i18n.T("label_max_preload_count"))
	maxPreloadInput.ForceDigits = true
	maxPreloadInput.OnSave = func() {
		config.Preload.MaxPreload, _ = strconv.Atoi(maxPreloadInput.Value)
		SaveConfig()
	}
	wholeContent.Add(maxPreloadInput)

	return wholeContent
}

func createDownloadSettingsContent() fyne.CanvasObject {
	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_download")))

	maxDownloadInput := widgets.NewInputWithSave(strconv.Itoa(config.Download.MaxDownload), i18n.T("label_max_parallel_download_count"))
	maxDownloadInput.ForceDigits = true
	maxDownloadInput.OnSave = func() {
		config.Download.MaxDownload, _ = strconv.Atoi(maxDownloadInput.Value)
		SaveConfig()
	}
	wholeContent.Add(maxDownloadInput)

	return wholeContent
}

func createCacheSettingsContent() fyne.CanvasObject {
	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_cache")))

	pathInput := widgets.NewInputWithSave(config.Cache.Path, i18n.T("label_cache_path"))
	pathInput.ForceDigits = true
	pathInput.OnSave = func() {
		config.Cache.Path = pathInput.Value
		SaveConfig()
	}
	wholeContent.Add(pathInput)

	maxCacheInput := widgets.NewInputWithSave(strconv.Itoa(config.Cache.MaxCacheSize), i18n.T("label_max_cache_size"))
	maxCacheInput.ForceDigits = true
	maxCacheInput.OnSave = func() {
		config.Preload.MaxPreload, _ = strconv.Atoi(pathInput.Value)
		SaveConfig()
	}
	maxCacheInput.BeforeSaveItems = []fyne.CanvasObject{widget.NewLabel("MB")}
	wholeContent.Add(maxCacheInput)

	return wholeContent
}
