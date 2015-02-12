package gmusic

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/mjibson/mog/_third_party/golang.org/x/oauth2"
	"github.com/mjibson/mog/codec"
	"github.com/mjibson/mog/codec/mpa"
	"github.com/mjibson/mog/protocol"
	"github.com/mjibson/mog/protocol/gmusic/gmusic"
)

func init() {
	gob.Register(new(GMusic))
	protocol.Register("gmusic", []string{"username", "password"}, New)
}

func New(params []string, token *oauth2.Token) (protocol.Instance, error) {
	if len(params) != 2 {
		return nil, fmt.Errorf("gmusic: bad params")
	}
	g, err := gmusic.Login(params[0], params[1])
	if err != nil {
		return nil, err
	}
	return &GMusic{
		GMusic: g,
	}, nil
}

func (g *GMusic) Info(id protocol.ID) (*codec.SongInfo, error) {
	s := g.Songs[id]
	if s == nil {
		for k := range g.Songs {
			fmt.Println(k)
		}
		panic("could not find: " + id)
		return nil, fmt.Errorf("could not find %v", id)
	}
	return s, nil
}

type GMusic struct {
	GMusic *gmusic.GMusic
	Tracks map[protocol.ID]*gmusic.Track
	Songs  protocol.SongList
}

func (g *GMusic) Key() string {
	return g.GMusic.DeviceID
}

func (g *GMusic) List() (protocol.SongList, []*protocol.Playlist, error) {
	if len(g.Songs) == 0 {
		return g.Refresh()
	}
	return g.Songs, nil, nil
}

func (g *GMusic) GetSong(id protocol.ID) (codec.Song, error) {
	f := g.Tracks[id]
	if f == nil {
		return nil, fmt.Errorf("missing %v", id)
	}
	return mpa.NewSong(func() (io.ReadCloser, int64, error) {
		log.Println("GMUSIC", id)
		r, err := g.GMusic.GetStream(string(id))
		if err != nil {
			return nil, 0, err
		}
		size, _ := strconv.ParseInt(f.EstimatedSize, 10, 64)
		return r.Body, size, nil
	})
}

func (g *GMusic) Refresh() (protocol.SongList, []*protocol.Playlist, error) {
	tracks := make(map[protocol.ID]*gmusic.Track)
	songs := make(protocol.SongList)
	log.Println("get gmusic tracks")
	trackList, err := g.GMusic.ListTracks()
	if err != nil {
		return nil, nil, err
	}
	log.Println("got gmusic tracks", len(trackList))
	for _, t := range trackList {
		tracks[protocol.ID(t.ID)] = t
		duration, _ := strconv.Atoi(t.DurationMillis)
		songs[protocol.ID(t.ID)] = &codec.SongInfo{
			Time:   time.Duration(duration) * time.Millisecond,
			Artist: t.Artist,
			Title:  t.Title,
			Album:  t.Album,
			Track:  t.TrackNumber,
		}
	}
	g.Songs = songs
	g.Tracks = tracks
	playlists, err := g.GMusic.ListPlaylists()
	if err != nil {
		return nil, nil, err
	}
	entries, err := g.GMusic.ListPlaylistEntries()
	if err != nil {
		return nil, nil, err
	}
	mp := make(map[string]*protocol.Playlist)
	for _, p := range playlists {
		mp[p.ID] = &protocol.Playlist{
			Name: p.Name,
		}
	}
	for _, e := range entries {
		p := mp[e.PlaylistId]
		if p == nil {
			continue
		}
		p.Songs = append(p.Songs, protocol.ID(e.ID))
	}
	var pls []*protocol.Playlist
	for _, p := range mp {
		pls = append(pls, p)
	}
	return songs, pls, err
}
