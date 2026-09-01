package third_parties

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/nfnt/resize"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func GetThumbnailImage(client *requesting.ClientProvider, url string, ctx context.Context) (image.Image, error) {
	logger.InfoLn("Downloading thumbnail from ", url)
	req, err := client.NewGetRequest(url, ctx)
	if err != nil {
		return nil, err
	}
	requesting.SetupHeader(req, url)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%wservice temporarily unavailable: status code %d", interactive.ErrTemporarilyUnavailable, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter, _ := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 32)
		return nil, utils.NewThrottledError(time.Duration(retryAfter) * time.Second)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%wresource unavailable: status code %d", interactive.ErrUnrecoverable, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "image/jpeg" {
		return nil, fmt.Errorf("%winvalid content type: %s", interactive.ErrUnrecoverable, resp.Header.Get("Content-Type"))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%wfailed to decode thumbnail: %w", interactive.ErrUnrecoverable, err)
	}

	return resize.Resize(320, 0, img, resize.Bilinear), nil
}
