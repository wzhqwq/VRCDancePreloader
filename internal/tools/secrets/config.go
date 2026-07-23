package secrets

type Config struct {
	YoutubeApiKey string
}

func (c Config) Validate(_ string) error {
	return nil
}

func DefaultConfig() (Config, error) {
	youtubeApiKey, err := getFromKeyring(YoutubeKeyUser)
	if err != nil {
		return Config{}, err
	}
	return Config{
		YoutubeApiKey: youtubeApiKey,
	}, nil
}
