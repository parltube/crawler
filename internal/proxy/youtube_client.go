package proxy

import (
	"fmt"
	"io"
	"strings"
	"sync"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/crispyLL/parltube-crawler/internal/structs"
)

// по сути профиль должен содержать это вот так YoutubeClient -> profile [john au fingerprint that is for him] ->
type YoutubeClient struct {
	httpClient tls_client.HttpClient
	lock       sync.RWMutex
}

func (y *YoutubeClient) GetChannel(youtubeId string) (*structs.YtChannel, error) {
	if youtubeId == "" || !strings.HasPrefix(youtubeId, "UC") || !strings.HasPrefix(youtubeId, "@") {
		return nil, fmt.Errorf("Бляха муха")
	}

	var baseString string
	if strings.HasPrefix(youtubeId, "@") {
		baseString = "https://youtube.com/"
	} else {
		baseString = "http://youtube.com/channel/"
	}

	resp, err := y.httpClient.Get(baseString + youtubeId)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	channel, err := ExtractChannelInfo(body)

	if err != nil {
		return nil, err
	}

	return channel, nil
}

func (y *YoutubeClient) GetVideo(videoId string) (*structs.VideoMetaData, error) {
	if len(videoId) != 11 {
		return nil, fmt.Errorf("video id must be 11 characters long")
	}

	resp, err := y.httpClient.Get("https://youtube.com/watch?v=" + videoId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

}

func (y *YoutubeClient) Close() error {
	y.lock.Lock()
	y.httpClient.CloseIdleConnections()
	y.lock.Unlock()
	return nil
}

func ExtractChannelInfo(body []byte) (*structs.YtChannel, error) {
	return
}

// checks if wether this client is banned, then sends exact error
func (y *YoutubeClient) CheckResponseForErrors(byte []body) error {

}

func NewYoutubeClient(httpClient tls_client.HttpClient) *YoutubeClient {
	return &YoutubeClient{httpClient: httpClient}
}
