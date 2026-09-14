package settings

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/cache_window"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/config_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/host"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func rangedInput(setting interactive.StatefulSetting[int], label string, minimum, maximum int64) *input.InputWithSave {
	i := input.NewInputWithSave(setting, label)
	i.Policy = input.NewPolicy(
		input.IntegerFilter(false),
		input.IntRangeValidator(minimum, maximum, false),
	)
	return i
}

func portInput(service interactive.StatefulService, setting interactive.StatefulSetting[int], label string) *input.InputWithRunner {
	i := input.NewInputWithRunner(service, setting, label)
	i.Policy = input.NewPolicy(
		input.IntegerFilter(false),
		input.IntRangeValidator(1024, 49151, false),
	)
	return i
}

func createHijackSettingsContent(cfg *config.Manager) fyne.CanvasObject {
	return container.NewVBox(
		container.NewHBox(
			widget.NewLabel(i18n.T("label_hijack")),
			container.NewCenter(button.NewTipButton("tip_on_hijack")),
		),
		portInput(host.MixedServer(), cfg.MixedServerPort(), i18n.T("label_hijack_proxy_port")),
		input.NewCheck(i18n.T("label_hijack_enable_https"), cfg.MixedServerEnableHttps()),
		config_widgets.NewMultiSelectSites(cfg.MixedServerInterceptedSites()),
	)
}

func newProxyInput(cfg *config.Manager, field string, label string) *input.InputWithTester {
	tester := requesting.GetTester(field)
	if tester == nil {
		return nil
	}
	return input.NewInputWithTester(tester, cfg.ProxyOfSite(field), label)
}

func createProxySettingsContent(cfg *config.Manager) fyne.CanvasObject {
	return container.NewVBox(
		container.NewHBox(
			widget.NewLabel(i18n.T("label_proxy")),
			container.NewCenter(button.NewTipButton("tip_on_proxy")),
		),
		newProxyInput(cfg, "pypydance-api", i18n.T("label_pypy_proxy")),
		newProxyInput(cfg, "wannadance-api", i18n.T("label_wanna_proxy")),
		newProxyInput(cfg, "dudu-fitdance-api", i18n.T("label_dudu_proxy")),
		newProxyInput(cfg, "bilibili", i18n.T("label_bili_proxy")),
		newProxyInput(cfg, "youtube-video", i18n.T("label_yt_video_proxy")),
		newProxyInput(cfg, "youtube-api", i18n.T("label_yt_api_proxy")),
		newProxyInput(cfg, "youtube-image", i18n.T("label_yt_image_proxy")),
		newProxyInput(cfg, "github-api", i18n.T("label_github_proxy")),
		newProxyInput(cfg, "github-assets", i18n.T("label_github_assets_proxy")),
	)
}

type youtubeApiGetAndSub struct {
	cfg *config.Manager
}

func (y *youtubeApiGetAndSub) Get() bool {
	return y.cfg.ThirdPartyBiliBiliMode().Get() == "api"
}

func (y *youtubeApiGetAndSub) Subscribe() *utils.EventSubscriber[bool] {
	return y.cfg.ThirdPartyYoutubeMode().SubscribeWhether(func(mode string) bool {
		return mode == "api"
	})
}

func createThirdPartySettingsContent(cfg *config.Manager) fyne.CanvasObject {
	modeOptions := []input.NamedOption[string]{
		{"disabled", i18n.T("option_via_disabled")},
		{"api", i18n.T("option_via_api")},
		{"ytdlp", i18n.T("option_via_ytdlp")},
	}

	return container.NewVBox(
		widget.NewLabel(i18n.T("label_third_parties")),
		input.NewHRadioGroup(i18n.T("label_yt_mode"), modeOptions, cfg.ThirdPartyYoutubeMode()),
		interactive_widgets.NewAvailableWhen(
			input.NewInputWithSave(cfg.SecretYoutubeAPIKey(), i18n.T("label_yt_api_key")),
			&youtubeApiGetAndSub{cfg},
		),
		input.NewHRadioGroup(i18n.T("label_bili_mode"), modeOptions, cfg.ThirdPartyBiliBiliMode()),
		config_widgets.NewAllowedResources(cfg),
	)
}

func createExecutableSettingsContent(cfg *config.Manager) fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel(i18n.T("label_executable")),
		input.NewCheck(i18n.T("label_executable_auto_check"), cfg.ExecutableAutoCheck()),
		config_widgets.NewDownloadableBinaryGui("ytdlp", cfg.ExecutableYtDlpPath()),
		config_widgets.NewDownloadableBinaryGui("deno", cfg.ExecutableDenoPath()),
	)
}

func createPreloadSettingsContent(cfg *config.Manager) fyne.CanvasObject {
	roomOptions := []input.NamedOption[string]{
		{"PyPyDance", "PyPyDance"},
		{"WannaDance", "WannaDance"},
		{"DuDuFitDance", "DuDuFitDance"},
	}
	roomOptionsWithFallback := []input.NamedOption[string]{
		{"PyPyDance", "PyPyDance"},
		{"DuDuFitDance", "DuDuFitDance"},
	}

	immutableCheck := widget.NewCheck(i18n.T("label_rejected_fallback"), nil)
	immutableCheck.Checked = true
	immutableCheck.Disable()

	return container.NewVBox(
		widget.NewLabel(i18n.T("label_preload")),
		rangedInput(cfg.PreloaderMaxCount(), i18n.T("label_max_preload_count"), 0, 10),
		input.WrapLabel(
			widgets.NewMultiSelect(roomOptions, cfg.PreloaderEnabledRooms()),
			i18n.T("label_preload_enabled_rooms"),
		),
		input.WrapLabel(
			widgets.NewMultiSelect(roomOptionsWithFallback, cfg.PreloaderYouTubeFallback()),
			i18n.T("label_preload_youtube_fallback"),
		),
		interactive_widgets.NewAvailableWhen(
			input.WrapLabel(
				container.NewVBox(
					immutableCheck,
					input.NewCheck(i18n.T("label_throttled_fallback"), cfg.PreloaderThrottledFallback()),
					input.NewCheck(i18n.T("label_low_speed_fallback"), cfg.PreloaderLowSpeedFallback()),
					interactive_widgets.NewAvailableWhen(
						input.NewCheck(i18n.T("label_high_fps_fallback"), cfg.PreloaderHighFramerateFallback()),
						interactive.NewReadonlyDerivedSetting(
							cfg.PreloaderYouTubeFallback(),
							func(rooms []string) bool {
								return lo.Contains(rooms, preloader.DuDuFitDanceRoomName)
							},
						),
					),
				),
				i18n.T("label_when_fallback"),
			),
			interactive.NewReadonlyDerivedSetting(
				cfg.PreloaderYouTubeFallback(),
				func(rooms []string) bool {
					return len(rooms) > 0
				},
			),
		),
	)
}

func createDownloadSettingsContent(cfg *config.Manager) fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel(i18n.T("label_download")),
		rangedInput(cfg.DownloaderMaxParallel(), i18n.T("label_max_parallel_download_count"), 1, 5),
	)
}

func createCacheSettingsContent(cfg *config.Manager) fyne.CanvasObject {
	formatOptions := []input.NamedOption[int]{
		{1, i18n.T("option_continuous")},
		{2, i18n.T("option_fragmented")},
	}

	maxCacheInput := rangedInput(cfg.MaxVideoCache(), i18n.T("label_max_cache_size"), 1, 1024*1024)
	maxCacheInput.InputAppendItems = []fyne.CanvasObject{widget.NewLabel("MB")}

	return container.NewVBox(
		container.NewHBox(
			widget.NewLabel(i18n.T("label_cache")),
			container.NewCenter(button.NewTipButton("tip_on_cache")),
		),
		input.NewInputWithSave(cfg.CachePath(), i18n.T("label_cache_path")),
		input.NewHRadioGroup(i18n.T("label_cache_format"), formatOptions, cfg.VideoFileFormat()),
		maxCacheInput,
		input.NewCheck(i18n.T("label_keep_favorites"), cfg.CacheKeepFavorites()),
		input.NewCheck(i18n.T("label_force_exp_check"), cfg.CacheForceExpiration()),
		widget.NewButton(i18n.T("btn_manage_cache"), func() {
			cache_window.OpenCacheWindow()
		}),
	)
}
