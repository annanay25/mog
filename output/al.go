// +build darwin

package output

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"golang.org/x/mobile/audio"
)

type al struct {
	pl   *audio.Player
	ch   chan []byte
	over []byte
	tick time.Duration
}

func get(sampleRate, channels int) (Output, error) {
	o := &al{
		ch:   make(chan []byte),
		tick: time.Second / time.Duration(sampleRate),
	}
	var err error
	var format audio.Format
	switch channels {
	case 1:
		format = audio.Mono16
	case 2:
		format = audio.Stereo16
	default:
		return nil, fmt.Errorf("unsupported num channels %v", channels)
	}
	o.pl, err = audio.NewPlayer(o, format, int64(sampleRate))
	if err != nil {
		return nil, err
	}
	o.pl.SetVolume(1)
	return o, nil
}

func (a *al) Push(samples []float32) {
	s := make([]byte, len(samples)*2)
	for i, v := range samples {
		binary.BigEndian.PutUint16(s[i*2:], uint16(float32(math.MaxUint16)*v))
	}
	a.ch <- s
}

// Read pulls out samples from the push channel as needed. It takes care
// of the cases where we need or have more or less samples than desired.
func (a *al) Read(out []byte) (int, error) {
	// Write previously saved samples.
	i := copy(out, a.over)
	defer func() {
		d := a.tick * time.Duration(i/2)
		fmt.Println("sleeping", d, i)
		time.Sleep(d)
		println("i", i, "of", len(out))
	}()
	a.over = a.over[i:]
Loop:
	for i < len(out) {
		select {
		case s := <-a.ch:
			n := copy(out[i:], s)
			if n < len(s) {
				// Save anything we didn't need this time.
				a.over = s[n:]
			}
			i += n
		default:
			break Loop
		}
	}
	return i, nil
}

func (a *al) Stop() {
	if err := a.pl.Stop(); err != nil {
		log.Println(err)
	}
}

func (a *al) Start() {
	go func() {
		if err := a.pl.Play(); err != nil {
			log.Println(err)
		}
	}()
}

func (a *al) Seek(int64, int) (int64, error) {
	println("seek")
	return 0, nil
}
