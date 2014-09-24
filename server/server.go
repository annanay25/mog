// Package server implements the mog protocol.
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go.net/websocket"

	"github.com/julienschmidt/httprouter"
	"github.com/mjibson/mog/codec"
	"github.com/mjibson/mog/output"
	"github.com/mjibson/mog/protocol"
)

const (
	DefaultAddr = ":6601"
)

func ListenAndServe(addr string) error {
	server := New()
	return server.ListenAndServe()
}

const (
	statePlay State = iota
	stateStop
	statePause
)

type State int

func (s State) String() string {
	switch s {
	case statePlay:
		return "play"
	case stateStop:
		return "stop"
	case statePause:
		return "pause"
	}
	return ""
}

type Playlist []SongID

type SongID struct {
	Protocol string
	ID       string
}

func ParseSongID(s string) (id SongID, err error) {
	sp := strings.SplitN(s, "|", 2)
	if len(sp) != 2 {
		return id, fmt.Errorf("bad songid: %v", s)
	}
	return SongID{sp[0], sp[1]}, nil
}

func (s SongID) String() string {
	return fmt.Sprintf("%s|%s", s.Protocol, s.ID)
}

func (s SongID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SongID) UnmarshalJSON(b []byte) error {
	var v [2]string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	s.Protocol = v[0]
	s.ID = v[1]
	return nil
}

type Server struct {
	Addr string // TCP address to listen on, ":6601"

	Songs      map[SongID]codec.Song
	State      State
	Playlist   Playlist
	PlaylistID int
	// Index of current song in the playlist.
	PlaylistIndex int
	SongID        SongID
	Song          codec.Song
	Info          codec.SongInfo
	Elapsed       time.Duration
	Error         string
	Repeat        bool
	Random        bool
	Protocols     map[string][]string

	songID int
	ch     chan command
	waitch chan struct{}
	lock   sync.Locker
}

func (srv *Server) wait() {
	srv.lock.Lock()
	if srv.waitch == nil {
		srv.waitch = make(chan struct{})
	}
	srv.lock.Unlock()
	<-srv.waitch
}

func (srv *Server) broadcast() {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.waitch == nil {
		return
	}
	close(srv.waitch)
	srv.waitch = nil
}

var dir = filepath.Join("server")

func New() *Server {
	srv := Server{
		ch:        make(chan command),
		lock:      new(sync.Mutex),
		Songs:     make(map[SongID]codec.Song),
		Protocols: make(map[string][]string),
	}
	go srv.audio()
	return &srv
}

func (srv *Server) GetMux() *http.ServeMux {
	router := httprouter.New()
	router.GET("/api/status", JSON(srv.Status))
	router.GET("/api/list", JSON(srv.List))
	router.GET("/api/playlist/change", JSON(srv.PlaylistChange))
	router.GET("/api/playlist/get", JSON(srv.PlaylistGet))
	router.GET("/api/protocol/update", JSON(srv.ProtocolUpdate))
	router.GET("/api/protocol/get", JSON(srv.ProtocolGet))
	router.GET("/api/protocol/list", JSON(srv.ProtocolList))
	router.GET("/api/song/info", JSON(srv.SongInfo))
	router.GET("/api/cmd/:cmd", JSON(srv.Cmd))
	fs := http.FileServer(http.Dir(dir))
	mux := http.NewServeMux()
	mux.Handle("/static/", fs)
	mux.HandleFunc("/", index)
	mux.Handle("/api/", router)
	mux.Handle("/ws/", websocket.Handler(srv.WebSocket))
	return mux
}

// ListenAndServe listens on the TCP network address srv.Addr and then calls
// Serve to handle requests on incoming connections. If srv.Addr is blank,
// ":6601" is used.
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = DefaultAddr
	}
	mux := srv.GetMux()
	log.Println("mog: listening on", addr)
	return http.ListenAndServe(addr, mux)
}

func (srv *Server) WebSocket(ws *websocket.Conn) {
	for {
		srv.wait()
		if err := websocket.JSON.Send(ws, srv.status()); err != nil {
			log.Println(err)
			break
		}
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(dir, "static", "index.html"))
}

func (srv *Server) audio() {
	var o output.Output
	var t chan interface{}
	var present bool
	var dur time.Duration
	srv.State = stateStop
	var next, stop, tick, play, pause, prev func()
	prev = func() {
		srv.PlaylistIndex--
		if srv.Elapsed < time.Second*3 {
			srv.PlaylistIndex--
		}
		next()
	}
	pause = func() {
		switch srv.State {
		case statePause, stateStop:
			t = make(chan interface{})
			close(t)
			tick()
		case statePlay:
			t = nil
			srv.State = statePause
		}
	}
	next = func() {
		log.Println("next")
		stop()
		play()
	}
	stop = func() {
		log.Println("stop")
		srv.State = stateStop
		t = nil
		srv.Song = nil
	}
	tick = func() {
		if srv.Elapsed > srv.Info.Time {
			stop()
		}
		if srv.Song == nil {
			if len(srv.Playlist) == 0 {
				log.Println("empty playlist")
				stop()
				return
			} else if srv.PlaylistIndex >= len(srv.Playlist) {
				if srv.Repeat {
					srv.PlaylistIndex = 0
				} else {
					log.Println("end of playlist")
					stop()
					return
				}
			}
			srv.SongID = srv.Playlist[srv.PlaylistIndex]
			srv.Song, present = srv.Songs[srv.SongID]
			srv.PlaylistIndex++
			if !present {
				return
			}
			sr, ch, err := srv.Song.Init()
			if err != nil {
				log.Fatal(err)
			}
			if o != nil {
				o.Dispose()
			}
			o, err = output.NewPort(sr, ch)
			if err != nil {
				log.Fatalf("mog: could not open audio (%v, %v): %v", sr, ch, err)
			}
			srv.Info = srv.Song.Info()
			fmt.Println("playing", srv.Info)
			srv.Elapsed = 0
			dur = time.Second / (time.Duration(sr))
			t = make(chan interface{})
			close(t)
			srv.State = statePlay
		}
		const expected = 4096
		next, err := srv.Song.Play(expected)
		if err == nil {
			srv.Elapsed += time.Duration(len(next)) * dur
			if len(next) > 0 {
				o.Push(next)
			}
		} else {
			log.Println(err)
		}
		if len(next) < expected || err != nil {
			stop()
		}
	}
	play = func() {
		log.Println("play")
		if srv.PlaylistIndex > len(srv.Playlist) {
			srv.PlaylistIndex = 0
		}
		tick()
	}
	go func() {
		for _ = range time.Tick(time.Millisecond * 250) {
			srv.broadcast()
		}
	}()
	for {
		select {
		case <-t:
			tick()
		case cmd := <-srv.ch:
			switch cmd {
			case cmdPlay:
				play()
			case cmdStop:
				stop()
			case cmdNext:
				next()
			case cmdPause:
				pause()
			case cmdPrev:
				prev()
			default:
				log.Fatal("unknown command")
			}
		}
	}
}

type command int

const (
	cmdPlay command = iota
	cmdStop
	cmdNext
	cmdPause
	cmdPrev
)

func JSON(h func(http.ResponseWriter, *http.Request, httprouter.Params) (interface{}, error)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		d, err := h(w, r, ps)
		if err != nil {
			serveError(w, err)
			return
		}
		if d == nil {
			return
		}
		b, err := json.Marshal(d)
		if err != nil {
			serveError(w, err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	}
}

func (srv *Server) Cmd(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	switch cmd := ps.ByName("cmd"); cmd {
	case "play":
		srv.ch <- cmdPlay
	case "stop":
		srv.ch <- cmdStop
	case "next":
		srv.ch <- cmdNext
	case "prev":
		srv.ch <- cmdPrev
	case "pause":
		srv.ch <- cmdPause
	default:
		return nil, fmt.Errorf("unknown command: %v", cmd)
	}
	return nil, nil
}

func (srv *Server) SongInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	var si []codec.SongInfo
	r.ParseForm()
	for _, s := range r.Form["song"] {
		id, err := ParseSongID(s)
		if err != nil {
			return nil, err
		}
		song, ok := srv.Songs[id]
		if !ok {
			return nil, fmt.Errorf("unknown song: %v", id)
		}
		si = append(si, song.Info())
	}
	return si, nil
}

func (srv *Server) PlaylistGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	return srv.Playlist, nil
}

func (srv *Server) ProtocolGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	return protocol.Get(), nil
}
func (srv *Server) ProtocolList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	return srv.Protocols, nil
}

func (srv *Server) ProtocolUpdate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	p := r.FormValue("protocol")
	params := r.Form["params"]
	songs, err := protocol.List(p, params)
	if err != nil {
		return nil, err
	}
	srv.Protocols[p] = params
	for id := range srv.Songs {
		if id.Protocol == p {
			delete(srv.Songs, id)
		}
	}
	for id, s := range songs {
		srv.Songs[SongID{Protocol: p, ID: id}] = s
	}
	return nil, nil
}

// Takes form values:
// * clear: if set to anything will clear playlist
// * remove/add: song ids
// Duplicate songs will not be added.
func (srv *Server) PlaylistChange(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	srv.PlaylistID++
	srv.PlaylistIndex = 0
	t := PlaylistChange{
		PlaylistId: srv.PlaylistID,
	}
	if len(r.Form["clear"]) > 0 {
		srv.Playlist = nil
		srv.ch <- cmdStop
	}
	m := make(map[SongID]int)
	for i, id := range srv.Playlist {
		m[id] = i
	}
	for _, rem := range r.Form["remove"] {
		sp := strings.SplitN(rem, "|", 2)
		if len(sp) != 2 {
			t.Error("bad id: %v", rem)
			continue
		}
		id := SongID{sp[0], sp[1]}
		if s, ok := srv.Songs[id]; !ok {
			t.Error("unknown id: %v", rem)
		} else if s == srv.Song {
			srv.ch <- cmdStop
		}
		delete(m, id)
	}
	for _, add := range r.Form["add"] {
		sp := strings.SplitN(add, "|", 2)
		if len(sp) != 2 {
			t.Error("bad id: %v", add)
			continue
		}
		id := SongID{sp[0], sp[1]}
		if _, ok := srv.Songs[id]; !ok {
			t.Error("unknown id: %v", add)
		}
		m[id] = len(m)
	}
	srv.Playlist = make(Playlist, len(m))
	for songid, index := range m {
		srv.Playlist[index] = songid
	}
	return &t, nil
}

type PlaylistChange struct {
	PlaylistId int
	Errors     []string
}

func (p *PlaylistChange) Error(format string, a ...interface{}) {
	p.Errors = append(p.Errors, fmt.Sprintf(format, a...))
}

func (s *Server) List(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	songs := make([]SongID, 0)
	for id := range s.Songs {
		songs = append(songs, id)
	}
	return songs, nil
}

func (s *Server) status() *Status {
	return &Status{
		Playlist: s.PlaylistID,
		State:    s.State,
		Song:     s.SongID,
		Elapsed:  s.Elapsed.Seconds(),
		Time:     s.Info.Time.Seconds(),
	}
}

func (s *Server) Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (interface{}, error) {
	return s.status(), nil
}

type Status struct {
	// Playlist ID.
	Playlist int
	// Playback state
	State State
	// Song ID.
	Song SongID
	// Elapsed time of current song in seconds.
	Elapsed float64
	// Duration of current song in seconds.
	Time float64
}

func serveError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}