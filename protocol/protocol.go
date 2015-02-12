package protocol

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mjibson/mog/_third_party/golang.org/x/oauth2"
	"github.com/mjibson/mog/codec"
)

type Protocol struct {
	*Params
	OAuth       *oauth2.Config
	newInstance func([]string, *oauth2.Token) (Instance, error)
}

type Params struct {
	Params   []string `json:",omitempty"`
	OAuthURL string   `json:",omitempty"`
}

type Instance interface {
	// Key returns a unique identifier for the instance.
	Key() string
	// List returns the list of available songs, possibly cached.
	List() (SongList, []*Playlist, error)
	// Refresh forces an update of the song list.
	Refresh() (SongList, []*Playlist, error)
	// Info returns information about one song.
	Info(ID) (*codec.SongInfo, error)
	// GetSong returns a playable song.
	GetSong(ID) (codec.Song, error)
}

type SongList map[ID]*codec.SongInfo

func (p *Protocol) NewInstance(params []string, token *oauth2.Token) (Instance, error) {
	return p.newInstance(params, token)
}

var protocols = make(map[string]*Protocol)

func Register(name string, params []string, newInstance func([]string, *oauth2.Token) (Instance, error)) {
	protocols[name] = &Protocol{
		Params: &Params{
			Params: params,
		},
		newInstance: newInstance,
	}
}

func RegisterOAuth(name string, config *oauth2.Config, newInstance func([]string, *oauth2.Token) (Instance, error)) {
	protocols[name] = &Protocol{
		Params: &Params{
			OAuthURL: config.AuthCodeURL(""),
		},
		OAuth:       config,
		newInstance: newInstance,
	}
}

func ByName(name string) (*Protocol, error) {
	p, ok := protocols[name]
	if !ok {
		return nil, fmt.Errorf("unknown protocol")
	}
	return p, nil
}

func Get() map[string]Params {
	m := make(map[string]Params)
	for n, p := range protocols {
		m[n] = *p.Params
	}
	return m
}

type ID string

func (id ID) ParseID() (path string, num int, err error) {
	sp := strings.SplitN(string(id), "-", 2)
	i, err := strconv.Atoi(sp[0])
	if len(sp) != 2 || err != nil {
		return "", 0, fmt.Errorf("bad format")
	}
	return sp[1], i, nil
}

type Playlist struct {
	Name  string
	Songs []ID
}
