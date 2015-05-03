package youtube

import (
	"encoding/gob"
	"fmt"
	"io"

	"github.com/mjibson/mog/_third_party/golang.org/x/oauth2"
	"github.com/mjibson/mog/codec"
	"github.com/mjibson/mog/protocol"
	"github.com/mjibson/mog/protocol/youtube"
)

func init() {
	protocol.Register("youtube", []string{"URL"}, New)
	gob.Register(new(Youtube))
}

func New(params []string, token *oauth2.Token) (protocol.Instance, error) {
	if len(params) != 1 {
		return nil, fmt.Errorf("expected one parameter")
	}
	y := Youtube{
		ID: params[0],
	}
	if _, err := y.Refresh(); err != nil {
		return nil, err
	}
	return y, nil
}

type Youtube struct {
	ID string

	Songs  protocol.SongList
	Tracks map[string]*track
}

func (y *Youtube) Key() string {
	return b.ID
}

func (y *Youtube) new() (yy *youtube, streamURL string, err error) {
	yy, err = youtube.New(y.ID)
	if err != nil {
		return
	}
	furl = yy.Formats["140"]
	if furl == "" {
		err = fmt.Errorf("youtube: could not find format 140")
		return
	}
	return
}

func (y *Youtube) Refresh() (protocol.SongList, error) {
	yy, furl, err := y.new()
	if err != nil {
		return nil, err
	}
	return songs, err
}

func (y *Youtube) info() *codec.SongInfo {
	return &codec.SongInfo{
		Title: y.ID,
	}
}

func (y *Youtube) List() (protocol.SongList, error) {
	return protocol.SongList{
		y.ID: y.info(),
	}, nil
}

func (y *Youtube) Refresh() (protocol.SongList, error) {
	return y.List()
}

func (y *Youtube) Info(string) (*codec.SongInfo, error) {
	return y.info(), nil
}

func (y *Youtube) GetSong(string) (codec.Song, error) {
	_, furl, err := y.new()
	if err != nil {
		return nil, err
	}

	return m4a.NewSong(func() (io.ReadCloser, int64, error) {
		resp, err := y.get()
		if err != nil {
			return nil, 0, err
		}
		y.Close()
		y.body = resp.Body
		return s, 0, nil
	})
}
