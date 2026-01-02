package settings

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/cache_window"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

func createHijackSettingsContent() fyne.CanvasObject {
	hijackConfig := config.GetHijackConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(container.NewHBox(
		widget.NewLabel(i18n.T("label_hijack")),
		container.NewCenter(button.NewTipButton("tip_on_hijack")),
	))

	wholeContent.Add(hijackConfig.HijackRunner.GetInput(i18n.T("label_hijack_proxy_port")))

	enableHttpsCb := widget.NewCheck(i18n.T("label_hijack_enable_https"), func(b bool) {
		if hijackConfig.EnableHttps == b {
			return
		}
		hijackConfig.UpdateEnableHttps(b)
	})
	enableHttpsCb.Checked = hijackConfig.EnableHttps
	wholeContent.Add(enableHttpsCb)

	wholeContent.Add(config.NewMultiSelectSites(hijackConfig.InterceptedSites))

	return wholeContent
}

func createProxySettingsContent() fyne.CanvasObject {
	proxyConfig := config.GetProxyConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(container.NewHBox(
		widget.NewLabel(i18n.T("label_proxy")),
		container.NewCenter(button.NewTipButton("tip_on_proxy")),
	))
	wholeContent.Add(proxyConfig.ProxyControllers["pypydance-api"].GetInput(i18n.T("label_pypy_proxy")))
	wholeContent.Add(proxyConfig.ProxyControllers["wannadance-api"].GetInput(i18n.T("label_wanna_proxy")))
	//TODO cancel comment after implemented youtube preloading
	//wholeContent.Add(proxyConfig.ProxyControllers["youtube-video"].GetInput(i18n.T("label_yt_video_proxy")))
	wholeContent.Add(proxyConfig.ProxyControllers["youtube-api"].GetInput(i18n.T("label_yt_api_proxy")))
	wholeContent.Add(proxyConfig.ProxyControllers["youtube-image"].GetInput(i18n.T("label_yt_image_proxy")))

	return wholeContent
}

func createKeySettingsContent() fyne.CanvasObject {
	keyConfig := config.GetKeyConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(container.NewHBox(
		widget.NewLabel(i18n.T("label_keys")),
		container.NewCenter(button.NewTipButton("tip_on_keys")),
	))

	youtubeKeyInput := input.NewInputWithSave(keyConfig.Youtube, i18n.T("label_yt_api_key"))
	youtubeKeyInput.OnSave = func() error {
		keyConfig.Youtube = youtubeKeyInput.Value
		config.SaveConfig()
		return nil
	}

	wholeContent.Add(youtubeKeyInput)
	return wholeContent
}

func createYoutubeSettingsContent() fyne.CanvasObject {
	proxyConfig := config.GetProxyConfig()
	youtubeConfig := config.GetYoutubeConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_youtube")))

	enableApiCheck := widget.NewCheck(i18n.T("label_yt_api_enable"), func(b bool) {
		if youtubeConfig.EnableApi == b {
			return
		}
		youtubeConfig.UpdateEnableApi(b)
		if b {
			proxyConfig.ProxyControllers["youtube-api"].TestIfNotOk()
		}
	})
	enableApiCheck.Checked = youtubeConfig.EnableApi
	wholeContent.Add(enableApiCheck)

	enableThumbnailCheck := widget.NewCheck(i18n.T("label_yt_thumbnail_enable"), func(b bool) {
		if youtubeConfig.EnableThumbnail == b {
			return
		}
		youtubeConfig.UpdateEnableThumbnail(b)
		if b {
			proxyConfig.ProxyControllers["youtube-image"].TestIfNotOk()
		}
	})
	enableThumbnailCheck.Checked = youtubeConfig.EnableThumbnail
	wholeContent.Add(enableThumbnailCheck)

	//TODO cancel comment after implemented youtube preloading
	//enableVideoCheck := widget.NewCheck(i18n.T("label_yt_video_enable"), func(b bool) {
	//	if config.Youtube.EnableVideo == b {
	//		return
	//	}
	//	config.Youtube.EnableVideo = b
	//	SaveConfig()
	//	if b && config.Proxy.ProxyControllers["youtube-video"].Status != ProxyStatusOk {
	//		config.Proxy.ProxyControllers["youtube-video"].Test()
	//	}
	//})
	//enableVideoCheck.Checked = config.Youtube.EnableVideo
	//wholeContent.Add(enableVideoCheck)

	return wholeContent
}

func createPreloadSettingsContent() fyne.CanvasObject {
	preloadConfig := config.GetPreloadConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_preload")))

	maxPreloadInput := input.NewInputWithSave(strconv.Itoa(preloadConfig.MaxPreload), i18n.T("label_max_preload_count"))
	maxPreloadInput.ForceDigits = true
	maxPreloadInput.OnSave = func() error {
		count, err := strconv.Atoi(maxPreloadInput.Value)
		if err != nil {
			return err
		}
		preloadConfig.UpdateMaxPreload(count)
		return nil
	}
	wholeContent.Add(maxPreloadInput)

	return wholeContent
}

func createDownloadSettingsContent() fyne.CanvasObject {
	downloadConfig := config.GetDownloadConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(widget.NewLabel(i18n.T("label_download")))

	maxDownloadInput := input.NewInputWithSave(strconv.Itoa(downloadConfig.MaxDownload), i18n.T("label_max_parallel_download_count"))
	maxDownloadInput.ForceDigits = true
	maxDownloadInput.OnSave = func() error {
		count, err := strconv.Atoi(maxDownloadInput.Value)
		if err != nil {
			return err
		}
		downloadConfig.UpdateMaxDownload(count)
		return nil
	}
	wholeContent.Add(maxDownloadInput)

	return wholeContent
}

func createCacheSettingsContent() fyne.CanvasObject {
	cacheConfig := config.GetCacheConfig()

	wholeContent := container.NewVBox()
	wholeContent.Add(container.NewHBox(
		widget.NewLabel(i18n.T("label_cache")),
		container.NewCenter(button.NewTipButton("tip_on_cache")),
	))

	pathInput := input.NewInputWithSave(cacheConfig.Path, i18n.T("label_cache_path"))
	pathInput.OnSave = func() error {
		cacheConfig.Path = pathInput.Value
		config.SaveConfig()
		return nil
	}
	wholeContent.Add(pathInput)

	formatOptions := []string{
		i18n.T("option_legacy"),
		i18n.T("option_continuous"),
		i18n.T("option_fragmented"),
	}

	formatLabel := canvas.NewText(i18n.T("label_cache_format"), theme.Color(theme.ColorNamePlaceHolder))
	formatLabel.TextSize = 12
	formatSelect := widget.NewRadioGroup(formatOptions, func(option string) {
		index := lo.IndexOf(formatOptions, option)
		if index == -1 {
			index = 0
		}
		cacheConfig.UpdateFileFormat(index)
	})
	formatSelect.Selected = formatOptions[cacheConfig.FileFormat]
	formatSelect.Horizontal = true
	wholeContent.Add(container.NewVBox(formatLabel, formatSelect))

	maxCacheInput := input.NewInputWithSave(strconv.Itoa(cacheConfig.MaxCacheSize), i18n.T("label_max_cache_size"))
	maxCacheInput.ForceDigits = true
	maxCacheInput.OnSave = func() error {
		size, err := strconv.Atoi(maxCacheInput.Value)
		if err != nil {
			return err
		}
		cacheConfig.UpdateMaxSize(size)
		return nil
	}
	maxCacheInput.InputAppendItems = []fyne.CanvasObject{widget.NewLabel("MB")}
	wholeContent.Add(maxCacheInput)

	keepFavoriteCheck := widget.NewCheck(i18n.T("label_keep_favorites"), func(b bool) {
		cacheConfig.UpdateKeepFavorites(b)
	})
	keepFavoriteCheck.Checked = cacheConfig.KeepFavorites
	wholeContent.Add(keepFavoriteCheck)

	manageBtn := widget.NewButton(i18n.T("btn_manage_cache"), func() {
		cache_window.OpenCacheWindow()
	})
	wholeContent.Add(manageBtn)

	return wholeContent
}
