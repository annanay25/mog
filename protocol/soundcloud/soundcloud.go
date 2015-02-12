package soundcloud

import (
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mjibson/mog/_third_party/code.google.com/p/google-api-go-client/googleapi"

	"github.com/mjibson/mog/_third_party/golang.org/x/oauth2"
	"github.com/mjibson/mog/codec"
	"github.com/mjibson/mog/codec/mpa"
	"github.com/mjibson/mog/protocol"
	"github.com/mjibson/mog/protocol/soundcloud/soundcloud"
)

var config *oauth2.Config
var oauthClientID string

func init() {
	gob.Register(new(Soundcloud))
}

func Init(clientID, clientSecret, redirect string) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirect + "soundcloud",
		Scopes:       []string{"non-expiring"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://soundcloud.com/connect",
			TokenURL: "https://api.soundcloud.com/oauth2/token",
		},
	}
	oauthClientID = clientID
	protocol.RegisterOAuth("soundcloud", config, New)
}

func (s *Soundcloud) getService() (*soundcloud.Service, *http.Client, error) {
	c := config.Client(oauth2.NoContext, s.Token)
	svc, err := soundcloud.New(c, s.Token)
	return svc, c, err
}

type Soundcloud struct {
	Token     *oauth2.Token
	Favorites map[protocol.ID]*soundcloud.Favorite
}

func New(params []string, token *oauth2.Token) (protocol.Instance, error) {
	if token == nil {
		return nil, fmt.Errorf("expected oauth token")
	}
	return &Soundcloud{
		Token: token,
	}, nil
}

func (s *Soundcloud) Key() string {
	return s.Token.AccessToken
}

func (s *Soundcloud) Info(id protocol.ID) (*codec.SongInfo, error) {
	f := s.Favorites[id]
	if f == nil {
		return nil, fmt.Errorf("could not find %v", id)
	}
	return toInfo(f), nil
}

func toInfo(f *soundcloud.Favorite) *codec.SongInfo {
	return &codec.SongInfo{
		Time:   time.Duration(f.Duration) * time.Millisecond,
		Artist: f.User.Username,
		Title:  f.Title,
	}
}

func (s *Soundcloud) SongList() protocol.SongList {
	m := make(protocol.SongList)
	for k, f := range s.Favorites {
		m[k] = toInfo(f)
	}
	return m
}

func (s *Soundcloud) List() (protocol.SongList, []*protocol.Playlist, error) {
	if len(s.Favorites) == 0 {
		return s.Refresh()
	}
	return s.SongList(), nil, nil
}

func (s *Soundcloud) GetSong(id protocol.ID) (codec.Song, error) {
	fmt.Println("SOUNDCLOUD", id)
	_, client, err := s.getService()
	if err != nil {
		return nil, err
	}
	f := s.Favorites[id]
	if f == nil {
		return nil, fmt.Errorf("bad id: %v", id)
	}
	return mpa.NewSong(func() (io.ReadCloser, int64, error) {
		res, err := client.Get(f.StreamURL + "?client_id=" + oauthClientID)
		if err != nil {
			return nil, 0, err
		}
		if err := googleapi.CheckResponse(res); err != nil {
			return nil, 0, err
		}
		return res.Body, 0, nil
	})
}

func (s *Soundcloud) Refresh() (protocol.SongList, []*protocol.Playlist, error) {
	service, _, err := s.getService()
	if err != nil {
		return nil, nil, err
	}
	favorites, err := service.Favorites().Do()
	if err != nil {
		return nil, nil, err
	}
	favs := make(map[protocol.ID]*soundcloud.Favorite)
	for _, f := range favorites {
		favs[protocol.ID(strconv.FormatInt(f.ID, 10))] = f
	}
	s.Favorites = favs
	return s.SongList(), nil, err
}
