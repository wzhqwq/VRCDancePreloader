package i18n

import (
	"embed"
	"log"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"gopkg.in/yaml.v3"

	"github.com/cloudfoundry/jibber_jabber"
	"github.com/eduardolat/goeasyi18n"
)

//go:embed translations/en/*.yaml
var enTranslationsFS embed.FS

//go:embed translations/zh_CN/*.yaml
var zhCNTranslationsFS embed.FS

//go:embed translations/date/*.yaml
var dateTranslationsFS embed.FS

var i18n *goeasyi18n.I18n
var lang string

type DateTranslations struct {
	MonthsFull []string `yaml:"months_full"`
	MonthsAbbr []string `yaml:"months_abbr"`
}

var dateTranslations DateTranslations

func Init() {
	localLang, err := jibber_jabber.DetectIETF()
	if err != nil {
		lang = "en"
	} else {
		lang = localLang
	}
	log.Println("Detected language:", lang)

	i18n = goeasyi18n.NewI18n()
	// Load the translations
	enTranslations, err := goeasyi18n.LoadFromYamlFS(enTranslationsFS, "translations/en/*.yaml")
	if err != nil {
		panic(err)
	}
	zhCNTranslations, err := goeasyi18n.LoadFromYamlFS(zhCNTranslationsFS, "translations/zh_CN/*.yaml")
	if err != nil {
		panic(err)
	}

	// Register the translations
	i18n.AddLanguage("en", enTranslations)
	i18n.AddLanguage("zh-CN", zhCNTranslations)

	if lang == "en" {
		file, err := dateTranslationsFS.ReadFile("translations/date/en.yaml")
		if err != nil {
			panic(err)
		}

		err = yaml.Unmarshal(file, &dateTranslations)
		if err != nil {
			panic(err)
		}
	}
}

func T(key string, options ...goeasyi18n.Options) string {
	return i18n.Translate(lang, key, options...)
}

func ParseMonth(m time.Month) string {
	if lang == "en" {
		return dateTranslations.MonthsAbbr[m]
	}
	return strconv.Itoa(int(m))
}

func ParseDate(date time.Time) map[string]string {
	year, month, day := date.Date()

	return map[string]string{
		"Year":  strconv.Itoa(year),
		"Month": ParseMonth(month),
		"Day":   strconv.Itoa(day),
		"Time":  date.Format("15:04"),
	}
}

func GetLangWrapping() fyne.TextWrap {
	if lang == "zh-CN" {
		return fyne.TextWrapBreak
	}
	return fyne.TextWrapWord
}
