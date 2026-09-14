package preloader

import "fmt"

const PyPyDanceRoomName = "PyPyDance"
const WannaDanceRoomName = "WannaDance"
const DuDuFitDanceRoomName = "DuDuFitDance"

type Config struct {
	MaxPreload   int      `yaml:"max-preload"`
	EnabledRooms []string `yaml:"enabled-rooms"`

	// PyPyDance, DuDuFitDance
	UseYoutubeFallback []string `yaml:"use-youtube-fallback"`

	ThrottledFallback     bool `yaml:"throttled-fallback"`
	LowSpeedFallback      bool `yaml:"low-speed-fallback"`
	HighFramerateFallback bool `yaml:"high-framerate-fallback"`
}

func (c Config) Validate(field string) error {
	if field == "" {
		if err := validateRooms(c.EnabledRooms); err != nil {
			return err
		}
		if err := validateCount(c.MaxPreload); err != nil {
			return err
		}
		if err := validateRoomsWithYouTubeFallback(c.UseYoutubeFallback); err != nil {
			return err
		}
		return nil
	}

	switch field {
	case "enabled-rooms":
		return validateRooms(c.EnabledRooms)
	case "max-preload-count":
		return validateCount(c.MaxPreload)
	case "use-youtube-fallback":
		return validateRoomsWithYouTubeFallback(c.UseYoutubeFallback)
	}

	return nil
}

func DefaultConfig() Config {
	return Config{
		EnabledRooms: []string{PyPyDanceRoomName, WannaDanceRoomName, DuDuFitDanceRoomName},
		MaxPreload:   2,
	}
}

func validateRooms(rooms []string) error {
	for _, room := range rooms {
		if room != PyPyDanceRoomName && room != WannaDanceRoomName && room != DuDuFitDanceRoomName {
			return fmt.Errorf("invalid room: %s", room)
		}
	}
	return nil
}

func validateRoomsWithYouTubeFallback(rooms []string) error {
	for _, room := range rooms {
		if room != PyPyDanceRoomName && room != DuDuFitDanceRoomName {
			return fmt.Errorf("invalid room: %s", room)
		}
	}
	return nil
}

func validateCount(count int) error {
	if count < 0 {
		return fmt.Errorf("invalid count: %d", count)
	}
	return nil
}
