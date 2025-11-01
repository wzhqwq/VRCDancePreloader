package requesting

import (
	"fmt"
	"log"
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type testCase struct {
	url    string
	useGet bool

	expectedStatus   int
	minContentLength int64
}

func requestClient(client *http.Client, tc testCase) (*http.Response, error) {
	if tc.useGet {
		return client.Get(tc.url)
	}
	return client.Head(tc.url)
}

func accessClient(client *http.Client, tc testCase) error {
	resp, err := requestClient(client, tc)
	if err != nil {
		return err
	}
	if tc.expectedStatus != 0 && resp.StatusCode != tc.expectedStatus {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if tc.minContentLength != 0 && resp.ContentLength < tc.minContentLength {
		return fmt.Errorf("unexpected content length: %s (too short)", utils.PrettyByteSize(resp.ContentLength))
	}
	return nil
}

func testClient(client *http.Client, serviceName string, tc testCase) (bool, string) {
	log.Printf("Testing %s client", serviceName)

	err := accessClient(client, tc)
	if err != nil {
		if client.Transport == nil {
			log.Printf("[Warning] Cannot connect to %s service, maybe you should configure proxy: %v", serviceName, err)
		} else {
			log.Printf("[Warning] Cannot connect to %s service through provided proxy: %v", serviceName, err)
		}
		return false, err.Error()
	}

	return true, ""
}

func videoTestCase(url string) testCase {
	return testCase{
		url: url,

		expectedStatus:   http.StatusOK,
		minContentLength: 1024 * 1024,
	}
}

func anonymousTestCase(url string) testCase {
	return testCase{
		url: url,

		expectedStatus: http.StatusOK,
	}
}

func authenticatedTestCase(url string) testCase {
	return testCase{
		url:    url,
		useGet: true,

		expectedStatus: http.StatusForbidden,
	}
}
