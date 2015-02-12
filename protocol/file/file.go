package file

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mjibson/mog/_third_party/golang.org/x/oauth2"
	"github.com/mjibson/mog/codec"
	"github.com/mjibson/mog/protocol"
)

func init() {
	protocol.Register("file", []string{"directory"}, New)
	gob.Register(new(File))
}

func New(params []string, token *oauth2.Token) (protocol.Instance, error) {
	if len(params) != 1 {
		return nil, fmt.Errorf("expected one parameter")
	}
	p, err := filepath.Abs(params[0])
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &File{
		Path:  p,
		Songs: make(protocol.SongList),
	}, nil
}

type File struct {
	Path  string
	Songs protocol.SongList
}

func (f *File) Key() string {
	return f.Path
}

func (f *File) Info(id protocol.ID) (*codec.SongInfo, error) {
	v := f.Songs[id]
	if v == nil {
		return nil, fmt.Errorf("could not find %v", id)
	}
	return v, nil
}

func (f *File) GetSong(id protocol.ID) (codec.Song, error) {
	path, num, err := id.ParseID()
	if err != nil {
		return nil, err
	}
	songs, _, err := codec.ByExtension(path, fileReader(path))
	if err != nil {
		return nil, err
	}
	return songs[num], nil
}

func (f *File) List() (protocol.SongList, []*protocol.Playlist, error) {
	if len(f.Songs) == 0 {
		return f.Refresh()
	}
	return f.Songs, nil, nil
}

func (f *File) Refresh() (protocol.SongList, []*protocol.Playlist, error) {
	songs := make(protocol.SongList)
	err := filepath.Walk(f.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		ss, _, err := codec.ByExtension(path, fileReader(path))
		if err != nil || len(ss) == 0 {
			return nil
		}
		for i, s := range ss {
			id := fmt.Sprintf("%v-%v", i, path)
			info, _ := s.Info()
			if info.Title == "" {
				title := filepath.Base(path)
				if len(ss) != 1 {
					title += fmt.Sprintf(":%v", i)
				}
				info.Title = title
			}
			if info.Album == "" {
				info.Album = filepath.Base(filepath.Dir(path))
			}
			songs[protocol.ID(id)] = &info
		}
		return nil
	})
	f.Songs = songs
	return songs, nil, err
}

func fileReader(path string) codec.Reader {
	return func() (io.ReadCloser, int64, error) {
		log.Println("open file", path)
		f, err := os.Open(path)
		if err != nil {
			return nil, 0, err
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, 0, err
		}
		return f, fi.Size(), nil
	}
}
